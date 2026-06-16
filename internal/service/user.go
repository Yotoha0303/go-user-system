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
	UpdateUserPasswordByUserID(ctx context.Context, db *gorm.DB, userID int64, oldPasswordHash, newPasswordHash string) error
	ListUser(ctx context.Context, db *gorm.DB, limit, offset int) (model.User, error)
	UserDisabled(ctx context.Context, db *gorm.DB, userID int64) error
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

func (daoUserStore) UpdateUserPasswordByUserID(ctx context.Context, db *gorm.DB, userID int64, oldPasswordHash, newPasswordHash string) error {
	return dao.UpdateUserPasswordByUserID(ctx, db, userID, oldPasswordHash, newPasswordHash)
}

func (daoUserStore) ListUser(ctx context.Context, db *gorm.DB, limit, offset int) (model.User, error) {
	return dao.ListUser(ctx, db, limit, offset)
}

func (daoUserStore) UserDisabled(ctx context.Context, db *gorm.DB, userID int64) error {
	return dao.UserDisabled(ctx, db, userID)
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

// FIX 修改用户密码后，需要禁用上一次登录时的 access_token ，防止密码被再次修改
func (s *UserService) UpdateUserPassword(ctx context.Context, userID int64, req request.UpdatePasswordRequest) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		user, err := s.store.GetUserByID(ctx, tx, userID)
		if err != nil {
			return ErrUserNotFound
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err == nil {
			return ErrUserPasswordNoDifference
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return ErrInvalidCredentials
		}
		if err := s.store.UpdateUserPasswordByUserID(ctx, s.db, userID, user.PasswordHash, string(passwordHash)); err != nil {
			return apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeUpdateUserPasswordFailed,
				"修改密码失败",
				err,
			)
		}

		return nil
	})
}

// TODO 查询所有用户-只有管理员才能够遍历用户
func (s *UserService) ListUser(ctx context.Context) (model.User, error) {
	var user model.User

	return user, nil
}

// TODO 禁用用户-只有管理员才能够禁用用户
func (s *UserService) UserDisabled(ctx context.Context, userID int64) error {

	return nil
}
