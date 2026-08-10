package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"go-user-system/internal/apperror"
	"go-user-system/internal/model"
	"go-user-system/internal/request"
	"go-user-system/internal/response"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	MinPasswordCharacters = 12
	MaxPasswordBytes      = 72
)

var dummyPasswordHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

func (s *UserService) ensureDB() error {
	if s == nil || s.db == nil {
		return ErrDatabaseNotInitialized
	}
	return nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < MinPasswordCharacters || len([]byte(password)) > MaxPasswordBytes {
		return ErrPasswordTooShortOrTooLong
	}
	return nil
}

func (s *UserService) Register(ctx context.Context, req request.RegisterRequest) error {

	username := strings.TrimSpace(req.Username)

	if len(username) < 3 {
		return ErrUsernameTooShort
	}

	if err := validatePassword(req.Password); err != nil {
		return err
	}

	if err := s.ensureDB(); err != nil {
		return err
	}

	userInfo, err := s.store.GetUserByUsername(ctx, s.db, username)
	if err == nil && userInfo != nil {
		return ErrUsernameAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRegisterFailed,
			"注册失败",
			err,
		)
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRegisterFailed,
			"注册失败",
			err,
		)
	}
	user := model.User{
		Username:     username,
		PasswordHash: string(hashBytes),
		Nickname:     username,
		Status:       model.UserStatusActive,
	}

	if s.rbacRepo == nil {
		if err := s.store.CreateUser(ctx, s.db, &user); err != nil {
			return apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeRegisterFailed,
				"注册失败",
				err,
			)
		}
		return nil
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.store.CreateUser(ctx, tx, &user); err != nil {
			return err
		}
		return s.rbacRepo.AssignRoleToUserByCode(ctx, tx, user.ID, model.RoleCodeUser)
	}); err != nil {
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRegisterFailed,
			"注册失败",
			err,
		)
	}
	return nil
}

func (s *UserService) Login(ctx context.Context, req request.LoginRequest) (*model.User, error) {
	username := strings.TrimSpace(req.Username)

	if err := s.ensureDB(); err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByUsername(ctx, s.db, username)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(req.Password))
			return nil, ErrInvalidCredentials
		}
		return nil, apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeLoginFailed,
			"登录错误",
			err,
		)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.Status != model.UserStatusActive {
		return nil, ErrInvalidCredentials
	}

	lastLoginAt := time.Now()
	if err := s.store.UpdateLastLoginAtByID(ctx, s.db, user.ID, lastLoginAt); err != nil {
		return nil, apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeLoginFailed,
			"登录错误",
			err,
		)
	}
	user.LastLoginAt = &lastLoginAt

	return user, nil
}

func (s *UserService) GetProfile(ctx context.Context, userID int64) (*model.User, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}

	if err := s.ensureDB(); err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByID(ctx, s.db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeGetProfileFailed,
			"获取用户信息失败",
			err,
		)
	}

	if user.Status != model.UserStatusActive {
		return nil, ErrUserDisabled
	}

	return user, nil
}

func (s *UserService) UpdateNickname(ctx context.Context, userID int64, nickname string) error {
	if userID <= 0 {
		return ErrInvalidUserID
	}

	nickname = strings.TrimSpace(nickname)

	if nickname == "" {
		return ErrNicknameEmpty
	}

	if len(nickname) > 64 {
		return ErrNicknameTooLong
	}

	if err := s.ensureDB(); err != nil {
		return err
	}

	user, err := s.store.GetUserByID(ctx, s.db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeUpdateNicknameFailed,
			"更改昵称失败",
			err,
		)
	}

	if user.Nickname == nickname {
		return nil
	}

	if user.Status != model.UserStatusActive {
		return ErrUserDisabled
	}

	if err := s.store.UpdateNicknameByID(ctx, s.db, userID, nickname); err != nil {
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeUpdateNicknameFailed,
			"更改昵称失败",
			err,
		)
	}
	return nil
}

func (s *UserService) UpdateUserPassword(ctx context.Context, userID int64, req request.UpdatePasswordRequest) error {
	if userID <= 0 {
		return ErrInvalidUserID
	}
	if err := s.ensureDB(); err != nil {
		return err
	}
	if req.OldPassword == req.NewPassword {
		return ErrUserPasswordNoDifference
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := s.store.GetUserByIDForUpdate(ctx, tx, userID)
		if err != nil {
			return ErrUserNotFound
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
			return ErrUserEnteredTheOldPasswordIncorrectly
		}

		if err := validatePassword(req.NewPassword); err != nil {
			return err
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return ErrInvalidCredentials
		}
		if err := s.store.UpdateUserPasswordByUserID(ctx, tx, userID, user.PasswordHash, string(passwordHash)); err != nil {
			return apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeUpdateUserPasswordFailed,
				"修改密码失败",
				err,
			)
		}

		if s.refreshRepo != nil {
			if err := s.refreshRepo.RevokeAllByUserID(ctx, tx, userID, time.Now(), model.RefreshTokenRevokedReasonPasswordChange); err != nil {
				return apperror.Wrap(
					http.StatusInternalServerError,
					response.CodeUpdateUserPasswordFailed,
					"修改密码失败",
					err,
				)
			}
		}

		return nil
	})
}
