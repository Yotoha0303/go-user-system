package service

import (
	"context"
	"errors"
	"go-user-system/internal/apperror"
	"go-user-system/internal/model"
	"go-user-system/internal/repository"
	"go-user-system/internal/response"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type AuthService struct {
	db          *gorm.DB
	refreshRepo repository.RefreshTokenRepository
	now         func() time.Time
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		db:          db,
		refreshRepo: repository.NewGormRefreshTokenRepository(),
		now:         time.Now,
	}
}

func (s *AuthService) ensureDB() error {
	if s == nil || s.db == nil {
		return ErrDatabaseNotInitialized
	}
	return nil
}

func (s *AuthService) StoreRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	if token == nil || token.UserID <= 0 || token.JTI == "" || token.TokenHash == "" {
		return ErrRefreshTokenInvalid
	}

	if err := s.refreshRepo.Create(ctx, s.db, token); err != nil {
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRefreshTokenInvalid,
			"保存 refresh token 失败",
			err,
		)
	}
	return nil
}

func (s *AuthService) RotateRefreshToken(ctx context.Context, userID int64, oldJTI string, oldHash string, next *model.RefreshToken) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	if userID <= 0 || oldJTI == "" || oldHash == "" || next == nil {
		return ErrRefreshTokenInvalid
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.refreshRepo.FindByJTIForUpdate(ctx, tx, oldJTI)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRefreshTokenInvalid
			}
			return apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeRefreshTokenInvalid,
				"读取 refresh token 失败",
				err,
			)
		}

		if current.UserID != userID || current.TokenHash != oldHash {
			return ErrRefreshTokenInvalid
		}
		if current.RevokedAt != nil {
			return ErrRefreshTokenRevoked
		}
		if !current.ExpiresAt.After(s.now()) {
			return ErrRefreshTokenExpired
		}

		next.UserID = userID
		if err := s.refreshRepo.Create(ctx, tx, next); err != nil {
			return apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeRefreshTokenInvalid,
				"创建新 refresh token 失败",
				err,
			)
		}

		if err := s.refreshRepo.RevokeByJTI(ctx, tx, oldJTI, s.now(), &next.JTI); err != nil {
			return apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeRefreshTokenInvalid,
				"吊销旧 refresh token 失败",
				err,
			)
		}

		return nil
	})
}

func (s *AuthService) RevokeRefreshToken(ctx context.Context, userID int64, jti string, tokenHash string) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	if userID <= 0 || jti == "" || tokenHash == "" {
		return ErrRefreshTokenInvalid
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.refreshRepo.FindByJTIForUpdate(ctx, tx, jti)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRefreshTokenInvalid
			}
			return apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeRefreshTokenInvalid,
				"读取 refresh token 失败",
				err,
			)
		}

		if current.UserID != userID || current.TokenHash != tokenHash {
			return ErrRefreshTokenInvalid
		}
		if current.RevokedAt != nil {
			return nil
		}

		if err := s.refreshRepo.RevokeByJTI(ctx, tx, jti, s.now(), nil); err != nil {
			return apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeRefreshTokenInvalid,
				"吊销 refresh token 失败",
				err,
			)
		}
		return nil
	})
}

func (s *AuthService) RevokeAllRefreshTokens(ctx context.Context, userID int64) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	if userID <= 0 {
		return ErrInvalidUserID
	}

	if err := s.refreshRepo.RevokeAllByUserID(ctx, s.db, userID, s.now()); err != nil {
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRefreshTokenInvalid,
			"吊销用户 refresh token 失败",
			err,
		)
	}
	return nil
}
