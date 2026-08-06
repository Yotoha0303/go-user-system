package service

import (
	"context"
	"errors"
	"go-user-system/internal/model"
	"testing"
	"time"

	"gorm.io/gorm"
)

type fakeRefreshTokenRepo struct {
	current       *model.RefreshToken
	findErr       error
	createErr     error
	revokeErr     error
	created       *model.RefreshToken
	revokedJTI    string
	replacedByJTI *string
}

func (r *fakeRefreshTokenRepo) Create(ctx context.Context, db *gorm.DB, token *model.RefreshToken) error {
	r.created = token
	return r.createErr
}

func (r *fakeRefreshTokenRepo) FindByJTIForUpdate(ctx context.Context, db *gorm.DB, jti string) (*model.RefreshToken, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.current == nil || r.current.JTI != jti {
		return nil, gorm.ErrRecordNotFound
	}
	return r.current, nil
}

func (r *fakeRefreshTokenRepo) RevokeByJTI(ctx context.Context, db *gorm.DB, jti string, revokedAt time.Time, replacedByJTI *string) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}
	r.revokedJTI = jti
	r.replacedByJTI = replacedByJTI
	r.current.RevokedAt = &revokedAt
	return nil
}

func (r *fakeRefreshTokenRepo) RevokeAllByUserID(ctx context.Context, db *gorm.DB, userID int64, revokedAt time.Time) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}
	if r.current != nil && r.current.UserID == userID {
		r.current.RevokedAt = &revokedAt
	}
	return nil
}

func newUnitAuthService(t *testing.T, repo *fakeRefreshTokenRepo) *AuthService {
	t.Helper()

	return &AuthService{
		db:          openServiceDryRunDB(t),
		refreshRepo: repo,
		now: func() time.Time {
			return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
		},
	}
}

func TestAuthServiceRotatesRefreshToken(t *testing.T) {
	repo := &fakeRefreshTokenRepo{
		current: &model.RefreshToken{
			UserID:    7,
			JTI:       "old-jti",
			TokenHash: "old-hash",
			ExpiresAt: time.Date(2026, 7, 14, 13, 0, 0, 0, time.UTC),
		},
	}
	authService := newUnitAuthService(t, repo)
	next := &model.RefreshToken{
		JTI:       "new-jti",
		TokenHash: "new-hash",
		ExpiresAt: time.Date(2026, 7, 14, 14, 0, 0, 0, time.UTC),
	}

	err := authService.RotateRefreshToken(context.Background(), 7, "old-jti", "old-hash", next)
	if err != nil {
		t.Fatalf("rotate refresh token failed: %v", err)
	}

	if repo.created == nil || repo.created.JTI != "new-jti" || repo.created.UserID != 7 {
		t.Fatalf("expected new refresh token to be created, got %+v", repo.created)
	}
	if repo.revokedJTI != "old-jti" {
		t.Fatalf("expected old token revoked, got %s", repo.revokedJTI)
	}
	if repo.replacedByJTI == nil || *repo.replacedByJTI != "new-jti" {
		t.Fatalf("expected replaced by new-jti, got %v", repo.replacedByJTI)
	}
}

func TestAuthServiceRejectsRepeatedRefreshToken(t *testing.T) {
	revokedAt := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	repo := &fakeRefreshTokenRepo{
		current: &model.RefreshToken{
			UserID:    7,
			JTI:       "old-jti",
			TokenHash: "old-hash",
			ExpiresAt: time.Date(2026, 7, 14, 13, 0, 0, 0, time.UTC),
			RevokedAt: &revokedAt,
		},
	}
	authService := newUnitAuthService(t, repo)

	err := authService.RotateRefreshToken(
		context.Background(),
		7,
		"old-jti",
		"old-hash",
		&model.RefreshToken{JTI: "new-jti", TokenHash: "new-hash"},
	)

	if !errors.Is(err, ErrRefreshTokenRevoked) {
		t.Fatalf("expected ErrRefreshTokenRevoked, got %v", err)
	}
}

func TestAuthServiceRejectsExpiredRefreshToken(t *testing.T) {
	repo := &fakeRefreshTokenRepo{
		current: &model.RefreshToken{
			UserID:    7,
			JTI:       "old-jti",
			TokenHash: "old-hash",
			ExpiresAt: time.Date(2026, 7, 14, 11, 0, 0, 0, time.UTC),
		},
	}
	authService := newUnitAuthService(t, repo)

	err := authService.RotateRefreshToken(
		context.Background(),
		7,
		"old-jti",
		"old-hash",
		&model.RefreshToken{JTI: "new-jti", TokenHash: "new-hash"},
	)

	if !errors.Is(err, ErrRefreshTokenExpired) {
		t.Fatalf("expected ErrRefreshTokenExpired, got %v", err)
	}
}
