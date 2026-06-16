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

func UpdateUserPasswordByUserID(ctx context.Context, db *gorm.DB, userID int64, oldPasswordHash, newPasswordHash string) error {
	return withContext(ctx, db).Where("password_hash = ? and id = ?", oldPasswordHash, userID).Model(&model.User{}).Update("password_hash", newPasswordHash).Error
}

func ListUser(ctx context.Context, db *gorm.DB, limit, offset int) (model.User, error) {
	var user model.User
	return user, withContext(ctx, db).Find(user).Limit(limit).Offset(offset).Error
}

func UserDisabled(ctx context.Context, db *gorm.DB, userID int64) error {
	return withContext(ctx, db).Where("id = ? and status = ?", userID, model.UserStatusActive).Model(&model.User{}).Update("status", model.UserStatusDisabled).Error
}
