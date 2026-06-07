package dao

import (
	"context"
	"go-user-system/internal/model"
	"time"

	"gorm.io/gorm"
)

func CreateUser(ctx context.Context, db *gorm.DB, user *model.User) error {
	return withContext(ctx, db).Create(user).Error
}

func GetUserByUsername(ctx context.Context, db *gorm.DB, username string) (*model.User, error) {
	var user model.User
	return &user, withContext(ctx, db).Where("username =?", username).First(&user).Error
}

func GetUserByID(ctx context.Context, db *gorm.DB, id int64) (*model.User, error) {
	var user model.User
	return &user, withContext(ctx, db).Where("id = ?", id).First(&user).Error
}

func UpdateNicknameByID(ctx context.Context, db *gorm.DB, id int64, nickname string) error {
	return withContext(ctx, db).Where("id = ?", id).Model(&model.User{}).Update("nickname", nickname).Error
}

func UpdateLastLoginAtByID(ctx context.Context, db *gorm.DB, id int64, lastLoginAt time.Time) error {
	return withContext(ctx, db).Where("id = ?", id).Model(&model.User{}).Update("last_login_at", lastLoginAt).Error
}

func withContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return db.WithContext(ctx)
}
