package service

import (
	"context"
	"go-user-system/internal/dao"
	"go-user-system/internal/model"
	"time"

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
	GetUserByIDForUpdate(ctx context.Context, db *gorm.DB, id int64) (*model.User, error)
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

func (daoUserStore) GetUserByIDForUpdate(ctx context.Context, db *gorm.DB, id int64) (*model.User, error) {
	return dao.GetUserByIDForUpdate(ctx, db, id)
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
