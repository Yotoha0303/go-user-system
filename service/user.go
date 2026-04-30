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

var (
	ErrUsernameEmpty    = errors.New("username is empty")
	ErrPasswordEmpty    = errors.New("password is empty")
	ErrUsernameTooShort = errors.New("username too short")
	ErrPasswordTooShort = errors.New("password too short")
	ErrUsernameExists   = errors.New("username already exists")
	ErrUserNotFound     = errors.New("username not found")
	ErrPasswordWrong    = errors.New("password incorrect")
	ErrUserDisabled     = errors.New("user disabled")
)

func Register(username, password string) error {

	username = strings.TrimSpace(username)
	if username == "" {
		return ErrUsernameEmpty
	}
	if password == "" {
		return ErrPasswordEmpty
	}
	if len(username) < 3 {
		return ErrUsernameTooShort
	}
	if len(password) < 6 {
		return ErrPasswordTooShort
	}

	existUser, err := dao.GetUserByUsername(global.DB, username)
	if err == nil && existUser != nil {
		return ErrUsernameExists
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

// 改、删；Service 层不应该伪装成 DAO；不使用db *gorm.DB为传入值
// func GetUserByUsername(username string) (*model.User, error) {
// 	var user model.User
// 	err := global.DB.Where("username = ?", username).First(&user).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &user, nil
// }

// 登录方法;返回账户信息
func Login(username, password string) (*model.User, error) {
	//增
	username = strings.TrimSpace(username)

	//增
	if username == "" {
		return nil, ErrUsernameEmpty
	}
	if password == "" {
		return nil, ErrPasswordEmpty
	}

	//1、账户是否存在；改
	user, err := dao.GetUserByUsername(global.DB, username)
	//改
	if err != nil {
		//增
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		//注意：返回错误
		return nil, err
	}

	if user.Status != model.UserStatusActive {
		return nil, ErrUserDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrPasswordWrong
	}
	//2、密码是否错误
	// dao.PasswordIsFailed(global.DB, username, password)

	//3、账户是否被禁用
	// dao.AccountIsDisabled(global.DB, username)

	return user, nil
}
