package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"go-user-system/internal/apperror"
	"go-user-system/internal/auth"
	"go-user-system/internal/authstate"
	"go-user-system/internal/dao"
	"go-user-system/internal/model"
	"go-user-system/internal/repository"
	"go-user-system/internal/response"

	"gorm.io/gorm"
)

type LoginRateLimit struct {
	AccountLimit int64
	IPLimit      int64
	Window       time.Duration
}

type authUserStore interface {
	GetUserByID(ctx context.Context, db *gorm.DB, id int64) (*model.User, error)
	GetUserByIDForUpdate(ctx context.Context, db *gorm.DB, id int64) (*model.User, error)
}

type daoAuthUserStore struct{}

func (daoAuthUserStore) GetUserByID(ctx context.Context, db *gorm.DB, id int64) (*model.User, error) {
	return dao.GetUserByID(ctx, db, id)
}

func (daoAuthUserStore) GetUserByIDForUpdate(ctx context.Context, db *gorm.DB, id int64) (*model.User, error) {
	return dao.GetUserByIDForUpdate(ctx, db, id)
}

type AuthService struct {
	db             *gorm.DB
	refreshRepo    repository.RefreshTokenRepository
	userStore      authUserStore
	stateStore     authstate.Store
	loginRateLimit LoginRateLimit
	now            func() time.Time
}

func NewAuthService(db *gorm.DB) *AuthService {
	return NewAuthServiceWithState(db, authstate.NewMemoryStore(), LoginRateLimit{
		AccountLimit: 5,
		IPLimit:      20,
		Window:       15 * time.Minute,
	})
}

func NewAuthServiceWithState(db *gorm.DB, stateStore authstate.Store, loginRateLimit LoginRateLimit) *AuthService {
	if stateStore == nil {
		stateStore = authstate.NewMemoryStore()
	}
	return &AuthService{
		db:             db,
		refreshRepo:    repository.NewGormRefreshTokenRepository(),
		userStore:      daoAuthUserStore{},
		stateStore:     stateStore,
		loginRateLimit: loginRateLimit,
		now:            time.Now,
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
	if token == nil || token.UserID <= 0 || token.JTI == "" || token.FamilyID == "" || token.TokenHash == "" {
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

func (s *AuthService) ValidateAccessToken(ctx context.Context, claims *auth.UserClaims) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	if claims == nil || claims.UserID <= 0 || claims.AuthVersion <= 0 || strings.TrimSpace(claims.JTI) == "" {
		return ErrAccessSessionInvalid
	}

	user, err := s.userStore.GetUserByID(ctx, s.db, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAccessSessionInvalid
		}
		return authStateUnavailable("读取用户认证状态失败", err)
	}
	if user.Status != model.UserStatusActive || user.AuthVersion != claims.AuthVersion {
		return ErrAccessSessionInvalid
	}

	revoked, err := s.stateStore.IsAccessTokenRevoked(ctx, claims.JTI)
	if err != nil {
		return authStateUnavailable("读取 access token 吊销状态失败", err)
	}
	if revoked {
		return ErrAccessSessionInvalid
	}
	return nil
}

func (s *AuthService) RotateRefreshToken(ctx context.Context, userID int64, authVersion int64, oldJTI string, oldHash string, next *model.RefreshToken) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	if userID <= 0 || authVersion <= 0 || oldJTI == "" || oldHash == "" || next == nil || next.JTI == "" || next.TokenHash == "" || !next.ExpiresAt.After(s.now()) {
		return ErrRefreshTokenInvalid
	}

	replayDetected := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := s.userStore.GetUserByIDForUpdate(ctx, tx, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRefreshTokenInvalid
			}
			return authStateUnavailable("读取用户认证状态失败", err)
		}
		if user.Status != model.UserStatusActive || user.AuthVersion != authVersion {
			return ErrRefreshTokenInvalid
		}

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
			if current.RevokedReason != nil && *current.RevokedReason == model.RefreshTokenRevokedReasonRotated && current.FamilyID != "" {
				if err := s.refreshRepo.RevokeFamily(
					ctx,
					tx,
					userID,
					current.FamilyID,
					s.now(),
					model.RefreshTokenRevokedReasonReplay,
				); err != nil {
					return apperror.Wrap(
						http.StatusInternalServerError,
						response.CodeRefreshTokenInvalid,
						"吊销 refresh token family 失败",
						err,
					)
				}
				replayDetected = true
				return nil
			}
			return ErrRefreshTokenRevoked
		}
		if !current.ExpiresAt.After(s.now()) {
			return ErrRefreshTokenExpired
		}

		next.UserID = userID
		next.FamilyID = current.FamilyID
		if next.FamilyID == "" {
			next.FamilyID = current.JTI
		}
		if err := s.refreshRepo.Create(ctx, tx, next); err != nil {
			return apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeRefreshTokenInvalid,
				"创建新 refresh token 失败",
				err,
			)
		}

		if err := s.refreshRepo.RevokeByJTI(
			ctx,
			tx,
			oldJTI,
			s.now(),
			model.RefreshTokenRevokedReasonRotated,
			&next.JTI,
		); err != nil {
			return apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeRefreshTokenInvalid,
				"吊销旧 refresh token 失败",
				err,
			)
		}

		return nil
	})
	if err != nil {
		return err
	}
	if replayDetected {
		return ErrRefreshTokenReplay
	}
	return nil
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

		if err := s.refreshRepo.RevokeByJTI(ctx, tx, jti, s.now(), model.RefreshTokenRevokedReasonLogout, nil); err != nil {
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

func (s *AuthService) RevokeAccessToken(ctx context.Context, jti string, expiresAt time.Time) error {
	if strings.TrimSpace(jti) == "" {
		return ErrAccessSessionInvalid
	}
	if s == nil || s.stateStore == nil {
		return authStateUnavailable("认证状态存储未初始化", nil)
	}
	ttl := expiresAt.Sub(s.now())
	if ttl <= 0 {
		return nil
	}
	if err := s.stateStore.RevokeAccessToken(ctx, jti, ttl); err != nil {
		return authStateUnavailable("吊销 access token 失败", err)
	}
	return nil
}

func (s *AuthService) RevokeAllRefreshTokens(ctx context.Context, userID int64) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	if userID <= 0 {
		return ErrInvalidUserID
	}

	if err := s.refreshRepo.RevokeAllByUserID(ctx, s.db, userID, s.now(), model.RefreshTokenRevokedReasonPasswordChange); err != nil {
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRefreshTokenInvalid,
			"吊销用户 refresh token 失败",
			err,
		)
	}
	return nil
}

func (s *AuthService) CheckLoginAllowed(ctx context.Context, account, ip string) (time.Duration, error) {
	if err := s.ensureRateLimiter(); err != nil {
		return 0, err
	}
	retryAfter, err := s.stateStore.CheckLoginLimit(
		ctx,
		normalizeAccount(account),
		strings.TrimSpace(ip),
		s.loginRateLimit.AccountLimit,
		s.loginRateLimit.IPLimit,
		s.loginRateLimit.Window,
	)
	if err != nil {
		return 0, authStateUnavailable("读取登录限流状态失败", err)
	}
	return retryAfter, nil
}

func (s *AuthService) RecordLoginFailure(ctx context.Context, account, ip string) (time.Duration, error) {
	if err := s.ensureRateLimiter(); err != nil {
		return 0, err
	}
	retryAfter, err := s.stateStore.RecordLoginFailure(
		ctx,
		normalizeAccount(account),
		strings.TrimSpace(ip),
		s.loginRateLimit.AccountLimit,
		s.loginRateLimit.IPLimit,
		s.loginRateLimit.Window,
	)
	if err != nil {
		return 0, authStateUnavailable("记录登录失败次数失败", err)
	}
	return retryAfter, nil
}

func (s *AuthService) ResetLoginFailures(ctx context.Context, account string) error {
	if err := s.ensureRateLimiter(); err != nil {
		return err
	}
	if err := s.stateStore.ResetLoginAccount(ctx, normalizeAccount(account)); err != nil {
		return authStateUnavailable("清理登录失败次数失败", err)
	}
	return nil
}

func (s *AuthService) ensureRateLimiter() error {
	if s == nil || s.stateStore == nil || s.loginRateLimit.AccountLimit <= 0 || s.loginRateLimit.IPLimit <= 0 || s.loginRateLimit.Window <= 0 {
		return authStateUnavailable("登录限流未初始化", nil)
	}
	return nil
}

func normalizeAccount(account string) string {
	return strings.ToLower(strings.TrimSpace(account))
}

func authStateUnavailable(message string, cause error) error {
	return apperror.Wrap(
		http.StatusServiceUnavailable,
		response.CodeAuthStateUnavailable,
		message,
		cause,
	)
}
