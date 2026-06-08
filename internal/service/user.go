package service

import (
	"context"
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
	db    *gorm.DB
	store userStore
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		db:    db,
		store: daoUserStore{},
	}
}

type userStore interface {
	CreateUser(ctx context.Context, db *gorm.DB, user *model.User) error
	GetUserByUsername(ctx context.Context, db *gorm.DB, username string) (*model.User, error)
	GetUserByID(ctx context.Context, db *gorm.DB, id int64) (*model.User, error)
	UpdateNicknameByID(ctx context.Context, db *gorm.DB, id int64, nickname string) error
	UpdateLastLoginAtByID(ctx context.Context, db *gorm.DB, id int64, lastLoginAt time.Time) error
}

type daoUserStore struct{}

func (daoUserStore) CreateUser(ctx context.Context, db *gorm.DB, user *model.User) error {
	return dao.CreateUser(ctx, db, user)
}

func (daoUserStore) GetUserByUsername(ctx context.Context, db *gorm.DB, username string) (*model.User, error) {
	return dao.GetUserByUsername(ctx, db, username)
}

func (daoUserStore) GetUserByID(ctx context.Context, db *gorm.DB, id int64) (*model.User, error) {
	return dao.GetUserByID(ctx, db, id)
}

func (daoUserStore) UpdateNicknameByID(ctx context.Context, db *gorm.DB, id int64, nickname string) error {
	return dao.UpdateNicknameByID(ctx, db, id, nickname)
}

func (daoUserStore) UpdateLastLoginAtByID(ctx context.Context, db *gorm.DB, id int64, lastLoginAt time.Time) error {
	return dao.UpdateLastLoginAtByID(ctx, db, id, lastLoginAt)
}

func (s *UserService) ensureDB() error {
	if s == nil || s.db == nil {
		return ErrDatabaseNotInitialized
	}
	return nil
}

func (s *UserService) Register(ctx context.Context, req request.RegisterRequest) error {

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

func (s *UserService) Login(ctx context.Context, req request.LoginRequest) (*model.User, error) {
	username := strings.TrimSpace(req.Username)

	if err := s.ensureDB(); err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByUsername(ctx, s.db, username)

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
