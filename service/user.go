package service

import (
	"errors"
	"strings"

	"go-user-system/dao"
	"go-user-system/global"
	"go-user-system/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Register(username, password string) error {

	// if username == "" || password == "" {
	// 	return errors.New("username or password is empty")
	// }

	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username is empty")
	}
	if password == "" {
		return errors.New("password is empty")
	}
	if len(username) < 3 {
		return errors.New("username too short")
	}
	if len(password) < 6 {
		return errors.New("password too short")
	}

	existUser, err := dao.GetUserByUsername(global.DB, username)
	if err == nil && existUser != nil {
		return errors.New("username already exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := model.User{
		Username:     username,
		PasswordHash: string(hashBytes),
		Nickname:     username,
		Status:       1,
	}
	return dao.CreateUser(global.DB, &user)
}

func GetUserByUsername(db *gorm.DB, username string) (*model.User, error) {
	var user model.User
	err := db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
