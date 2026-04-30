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

// 登录密码是否存在错误
func PasswordIsFailed(db *gorm.DB, username, password string) (bool, error) {
	err := db.Where("username =?", username).Where("password =?", password).Error
	if err != nil {
		return false, err
	}
	return true, nil
}

// 账户是否被禁用(未完成)
func AccountIsDisabled(db *gorm.DB, username string) (bool, error) {
	//1、根据用户名查找账户信息
	err := db.Where("username =?", username).Error
	if err != nil {
		return false, err
	}
	//2、根据用户信息，查找该账户是否被禁用

	return true, nil
}
