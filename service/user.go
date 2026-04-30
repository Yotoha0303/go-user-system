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

// 登录方法;返回账户信息
func Login(username, password string) error {
	//1、账户是否存在
	_, err := dao.GetUserByUsername(global.DB, username)
	if err != nil {
		errors.New("username is not exist")
	}
	//2、密码是否错误
	dao.PasswordIsFailed(global.DB, username, password)

	//3、账户是否被禁用
	dao.AccountIsDisabled(global.DB, username)

	return nil
}
