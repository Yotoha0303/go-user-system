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
