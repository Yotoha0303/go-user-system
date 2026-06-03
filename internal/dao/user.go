package dao

import (
	"go-user-system/internal/model"

	"gorm.io/gorm"
)

func CreateUser(db *gorm.DB, user *model.User) error {
	return db.Create(user).Error
}

func GetUserByUsername(db *gorm.DB, username string) (*model.User, error) {
	var user model.User
	return &user, db.Where("username =?", username).First(&user).Error
}

func GetUserByID(db *gorm.DB, id int64) (*model.User, error) {
	var user model.User
	return &user, db.Where("id = ?", id).First(&user).Error
}

func UpdateNicknameByID(db *gorm.DB, id int64, nickname string) error {
	return db.Where("id = ?", id).Model(&model.User{}).Update("nickname", nickname).Error
}
