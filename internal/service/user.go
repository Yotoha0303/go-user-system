package service

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"go-user-system/internal/apperror"
	"go-user-system/internal/dao"
	"go-user-system/internal/model"
	"go-user-system/internal/request"
	"go-user-system/internal/response"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) ensureDB() error {
	if s == nil || s.db == nil {
		return ErrDatabaseNotInitialized
	}
	return nil
}

func (s *UserService) Register(req request.RegisterRequest) error {

	username := strings.TrimSpace(req.Username)

	if len(username) < 3 {
		return ErrUsernameTooShort
	}

	if len(req.Password) < 6 {
		return ErrPasswordTooShort
	}

	if err := s.ensureDB(); err != nil {
		return err
	}

	userInfo, err := dao.GetUserByUsername(s.db, username)
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

	if err := dao.CreateUser(s.db, &user); err != nil {
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRegisterFailed,
			"注册失败",
			err,
		)
	}
	return nil
}

func (s *UserService) Login(req request.LoginRequest) (*model.User, error) {
	username := strings.TrimSpace(req.Username)

	if err := s.ensureDB(); err != nil {
		return nil, err
	}

	user, err := dao.GetUserByUsername(s.db, username)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeLoginFailed,
			"登录错误",
			err,
		)
	}

	if user.Status != model.UserStatusActive {
		return nil, ErrUserDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	lastLoginAt := time.Now()
	if err := dao.UpdateLastLoginAtByID(s.db, user.ID, lastLoginAt); err != nil {
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

func (s *UserService) GetProfile(userID int64) (*model.User, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}

	if err := s.ensureDB(); err != nil {
		return nil, err
	}

	user, err := dao.GetUserByID(s.db, userID)
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

func (s *UserService) UpdateNickname(userID int64, nickname string) error {
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

	user, err := dao.GetUserByID(s.db, userID)
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

	if err := dao.UpdateNicknameByID(s.db, userID, nickname); err != nil {
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeUpdateNicknameFailed,
			"更改昵称失败",
			err,
		)
	}
	return nil
}
