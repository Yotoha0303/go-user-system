package service

import (
	"context"
	"errors"
	"go-user-system/internal/auth"
	"go-user-system/internal/authstate"
	"go-user-system/internal/model"
	"go-user-system/internal/response"
	"net/http"
	"sync"
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
	revokedReason string
	revokedFamily string
	replacedByJTI *string
}

type lockingRefreshTokenRepo struct {
	mu            sync.Mutex
	current       model.RefreshToken
	created       []*model.RefreshToken
	familyRevoked bool
}

func (r *lockingRefreshTokenRepo) Create(ctx context.Context, db *gorm.DB, token *model.RefreshToken) error {
	r.created = append(r.created, token)
	return nil
}

func (r *lockingRefreshTokenRepo) FindByJTIForUpdate(ctx context.Context, db *gorm.DB, jti string) (*model.RefreshToken, error) {
	r.mu.Lock()
	if r.current.JTI != jti {
		r.mu.Unlock()
		return nil, gorm.ErrRecordNotFound
	}
	return &r.current, nil
}

func (r *lockingRefreshTokenRepo) RevokeByJTI(ctx context.Context, db *gorm.DB, jti string, revokedAt time.Time, reason string, replacedByJTI *string) error {
	r.current.RevokedAt = &revokedAt
	r.current.RevokedReason = &reason
	r.current.ReplacedByJTI = replacedByJTI
	r.mu.Unlock()
	return nil
}

func (r *lockingRefreshTokenRepo) RevokeFamily(ctx context.Context, db *gorm.DB, userID int64, familyID string, revokedAt time.Time, reason string) error {
	r.familyRevoked = true
	for _, token := range r.created {
		token.RevokedAt = &revokedAt
		token.RevokedReason = &reason
	}
	r.mu.Unlock()
	return nil
}

func (r *lockingRefreshTokenRepo) RevokeAllByUserID(ctx context.Context, db *gorm.DB, userID int64, revokedAt time.Time, reason string) error {
	return nil
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

func (r *fakeRefreshTokenRepo) RevokeByJTI(ctx context.Context, db *gorm.DB, jti string, revokedAt time.Time, reason string, replacedByJTI *string) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}
	r.revokedJTI = jti
	r.revokedReason = reason
	r.replacedByJTI = replacedByJTI
	r.current.RevokedAt = &revokedAt
	r.current.RevokedReason = &reason
	return nil
}

func (r *fakeRefreshTokenRepo) RevokeFamily(ctx context.Context, db *gorm.DB, userID int64, familyID string, revokedAt time.Time, reason string) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}
	r.revokedFamily = familyID
	r.revokedReason = reason
	return nil
}

func (r *fakeRefreshTokenRepo) RevokeAllByUserID(ctx context.Context, db *gorm.DB, userID int64, revokedAt time.Time, reason string) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}
	if r.current != nil && r.current.UserID == userID {
		r.current.RevokedAt = &revokedAt
		r.current.RevokedReason = &reason
	}
	return nil
}

type fakeAuthUserStore struct {
	user *model.User
	err  error
}

type fakeAuthStateStore struct {
	revoked bool
	err     error
}

func (s fakeAuthStateStore) RevokeAccessToken(context.Context, string, time.Duration) error {
	return s.err
}

func (s fakeAuthStateStore) IsAccessTokenRevoked(context.Context, string) (bool, error) {
	return s.revoked, s.err
}

func (s fakeAuthStateStore) CheckLoginLimit(context.Context, string, string, int64, int64, time.Duration) (time.Duration, error) {
	return 0, s.err
}

func (s fakeAuthStateStore) RecordLoginFailure(context.Context, string, string, int64, int64, time.Duration) (time.Duration, error) {
	return 0, s.err
}

func (s fakeAuthStateStore) ResetLoginAccount(context.Context, string) error {
	return s.err
}

func (s fakeAuthStateStore) Ping(context.Context) error {
	return s.err
}

func (s fakeAuthStateStore) Close() error {
	return nil
}

func (s fakeAuthUserStore) GetUserByID(ctx context.Context, db *gorm.DB, id int64) (*model.User, error) {
	return s.user, s.err
}

func (s fakeAuthUserStore) GetUserByIDForUpdate(ctx context.Context, db *gorm.DB, id int64) (*model.User, error) {
	return s.user, s.err
}

func newUnitAuthService(t *testing.T, repo *fakeRefreshTokenRepo) *AuthService {
	t.Helper()

	return &AuthService{
		db:          openServiceDryRunDB(t),
		refreshRepo: repo,
		userStore: fakeAuthUserStore{user: &model.User{
			ID:          7,
			Status:      model.UserStatusActive,
			AuthVersion: 1,
		}},
		stateStore: authstate.NewMemoryStore(),
		loginRateLimit: LoginRateLimit{
			AccountLimit: 5,
			IPLimit:      20,
			Window:       time.Minute,
		},
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
			FamilyID:  "family-1",
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

	err := authService.RotateRefreshToken(context.Background(), 7, 1, "old-jti", "old-hash", next)
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
	if next.FamilyID != "family-1" {
		t.Fatalf("expected inherited family id, got %q", next.FamilyID)
	}
	if repo.revokedReason != model.RefreshTokenRevokedReasonRotated {
		t.Fatalf("expected rotated reason, got %q", repo.revokedReason)
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
		1,
		"old-jti",
		"old-hash",
		&model.RefreshToken{JTI: "new-jti", TokenHash: "new-hash", ExpiresAt: time.Date(2026, 7, 14, 14, 0, 0, 0, time.UTC)},
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
		1,
		"old-jti",
		"old-hash",
		&model.RefreshToken{JTI: "new-jti", TokenHash: "new-hash", ExpiresAt: time.Date(2026, 7, 14, 14, 0, 0, 0, time.UTC)},
	)

	if !errors.Is(err, ErrRefreshTokenExpired) {
		t.Fatalf("expected ErrRefreshTokenExpired, got %v", err)
	}
}

func TestAuthServiceRevokesFamilyOnRefreshReplay(t *testing.T) {
	revokedAt := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	reason := model.RefreshTokenRevokedReasonRotated
	repo := &fakeRefreshTokenRepo{
		current: &model.RefreshToken{
			UserID:        7,
			JTI:           "old-jti",
			FamilyID:      "family-1",
			TokenHash:     "old-hash",
			ExpiresAt:     time.Date(2026, 7, 14, 13, 0, 0, 0, time.UTC),
			RevokedAt:     &revokedAt,
			RevokedReason: &reason,
		},
	}
	authService := newUnitAuthService(t, repo)

	err := authService.RotateRefreshToken(
		context.Background(),
		7,
		1,
		"old-jti",
		"old-hash",
		&model.RefreshToken{JTI: "new-jti", TokenHash: "new-hash", ExpiresAt: time.Date(2026, 7, 14, 14, 0, 0, 0, time.UTC)},
	)

	if !errors.Is(err, ErrRefreshTokenReplay) {
		t.Fatalf("expected ErrRefreshTokenReplay, got %v", err)
	}
	if repo.revokedFamily != "family-1" {
		t.Fatalf("expected family-1 revoked, got %q", repo.revokedFamily)
	}
	if repo.revokedReason != model.RefreshTokenRevokedReasonReplay {
		t.Fatalf("expected replay reason, got %q", repo.revokedReason)
	}
}

func TestAuthServiceValidatesActiveAccessSession(t *testing.T) {
	authService := newUnitAuthService(t, &fakeRefreshTokenRepo{})

	err := authService.ValidateAccessToken(context.Background(), &auth.UserClaims{
		UserID:      7,
		AuthVersion: 1,
		JTI:         "access-jti",
	})
	if err != nil {
		t.Fatalf("expected active session, got %v", err)
	}
}

func TestAuthServiceRejectsDisabledAndStaleAccessSessions(t *testing.T) {
	tests := []struct {
		name        string
		user        *model.User
		authVersion int64
	}{
		{
			name:        "disabled user",
			user:        &model.User{ID: 7, Status: model.UserStatusDisabled, AuthVersion: 1},
			authVersion: 1,
		},
		{
			name:        "stale auth version",
			user:        &model.User{ID: 7, Status: model.UserStatusActive, AuthVersion: 2},
			authVersion: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authService := newUnitAuthService(t, &fakeRefreshTokenRepo{})
			authService.userStore = fakeAuthUserStore{user: test.user}

			err := authService.ValidateAccessToken(context.Background(), &auth.UserClaims{
				UserID:      7,
				AuthVersion: test.authVersion,
				JTI:         "access-jti",
			})
			if !errors.Is(err, ErrAccessSessionInvalid) {
				t.Fatalf("expected ErrAccessSessionInvalid, got %v", err)
			}
		})
	}
}

func TestAuthServiceRejectsRevokedAccessSession(t *testing.T) {
	authService := newUnitAuthService(t, &fakeRefreshTokenRepo{})
	authService.stateStore = fakeAuthStateStore{revoked: true}

	err := authService.ValidateAccessToken(context.Background(), &auth.UserClaims{
		UserID:      7,
		AuthVersion: 1,
		JTI:         "access-jti",
	})
	if !errors.Is(err, ErrAccessSessionInvalid) {
		t.Fatalf("expected ErrAccessSessionInvalid, got %v", err)
	}
}

func TestAuthServiceFailsClosedWhenStateStoreIsUnavailable(t *testing.T) {
	authService := newUnitAuthService(t, &fakeRefreshTokenRepo{})
	authService.stateStore = fakeAuthStateStore{err: errors.New("redis unavailable")}

	err := authService.ValidateAccessToken(context.Background(), &auth.UserClaims{
		UserID:      7,
		AuthVersion: 1,
		JTI:         "access-jti",
	})
	assertServiceAppError(t, err, http.StatusServiceUnavailable, response.CodeAuthStateUnavailable)
}

func TestAuthServiceRejectsRefreshForDisabledUser(t *testing.T) {
	repo := &fakeRefreshTokenRepo{current: &model.RefreshToken{
		UserID:    7,
		JTI:       "old-jti",
		FamilyID:  "family-1",
		TokenHash: "old-hash",
		ExpiresAt: time.Date(2026, 7, 14, 13, 0, 0, 0, time.UTC),
	}}
	authService := newUnitAuthService(t, repo)
	authService.userStore = fakeAuthUserStore{user: &model.User{
		ID:          7,
		Status:      model.UserStatusDisabled,
		AuthVersion: 1,
	}}

	err := authService.RotateRefreshToken(
		context.Background(),
		7,
		1,
		"old-jti",
		"old-hash",
		&model.RefreshToken{JTI: "new-jti", TokenHash: "new-hash", ExpiresAt: time.Date(2026, 7, 14, 14, 0, 0, 0, time.UTC)},
	)
	if !errors.Is(err, ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid, got %v", err)
	}
	if repo.created != nil {
		t.Fatalf("expected no successor token, got %+v", repo.created)
	}
}

func TestAuthServiceConcurrentRefreshAllowsOneSuccess(t *testing.T) {
	repo := &lockingRefreshTokenRepo{current: model.RefreshToken{
		UserID:    7,
		JTI:       "old-jti",
		FamilyID:  "family-1",
		TokenHash: "old-hash",
		ExpiresAt: time.Date(2026, 7, 14, 13, 0, 0, 0, time.UTC),
	}}
	authService := &AuthService{
		db:          openServiceDryRunDB(t),
		refreshRepo: repo,
		userStore: fakeAuthUserStore{user: &model.User{
			ID:          7,
			Status:      model.UserStatusActive,
			AuthVersion: 1,
		}},
		now: func() time.Time {
			return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
		},
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	for _, nextJTI := range []string{"next-jti-1", "next-jti-2"} {
		nextJTI := nextJTI
		go func() {
			<-start
			results <- authService.RotateRefreshToken(
				context.Background(),
				7,
				1,
				"old-jti",
				"old-hash",
				&model.RefreshToken{
					JTI:       nextJTI,
					TokenHash: nextJTI + "-hash",
					ExpiresAt: time.Date(2026, 7, 14, 14, 0, 0, 0, time.UTC),
				},
			)
		}()
	}
	close(start)

	successes := 0
	replays := 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrRefreshTokenReplay):
			replays++
		default:
			t.Fatalf("unexpected refresh result: %v", err)
		}
	}
	if successes != 1 || replays != 1 {
		t.Fatalf("expected one success and one replay, successes=%d replays=%d", successes, replays)
	}
	if !repo.familyRevoked || len(repo.created) != 1 || repo.created[0].RevokedReason == nil || *repo.created[0].RevokedReason != model.RefreshTokenRevokedReasonReplay {
		t.Fatalf("expected the successor family token revoked, repo=%+v", repo)
	}
}
