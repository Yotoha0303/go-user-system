package dao

import (
	"go-user-system/model"

	"gorm.io/gorm"
)

func CreateUser(db *gorm.DB, user *model.User) error {
	return db.Create(user).Error
}

func GetUserByUsername(db *gorm.DB, username string) (*model.User, error) {
	var user model.User
	err := db.Where("username =?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByID(db *gorm.DB, id int64) (*model.User, error) {
	var user model.User
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func UpdateNicknameByID(db *gorm.DB, id int64, nickname string) error {

	return db.Model(&model.User{}).Where("id = ?", id).Update("nickname", nickname).Error
}
