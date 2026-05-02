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
	ErrNicknameEmpty    = errors.New("nickname is empty")
	ErrNicknameTooLong  = errors.New("nickname too long")
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
		Status:       model.UserStatusActive,
	}
	return dao.CreateUser(global.DB, &user)
}

func Login(username, password string) (*model.User, error) {
	username = strings.TrimSpace(username)

	if username == "" {
		return nil, ErrUsernameEmpty
	}
	if password == "" {
		return nil, ErrPasswordEmpty
	}

	user, err := dao.GetUserByUsername(global.DB, username)

	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if user.Status != model.UserStatusActive {
		return nil, ErrUserDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrPasswordWrong
	}

	return user, nil
}

func GetProfile(userID int64) (*model.User, error) {
	if userID == 0 {
		return nil, errors.New("userID is not empty")
	}

	user, err := dao.GetUserByID(global.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func UpdateNickname(userID int64, nickname string) error {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return ErrNicknameEmpty
	}
	if len(nickname) > 64 {
		return ErrNicknameTooLong
	}

	user, err := dao.GetUserByID(global.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if user.Status != model.UserStatusActive {
		return ErrUserDisabled
	}

	return dao.UpdateNicknameByID(global.DB, userID, nickname)
}
