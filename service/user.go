package service

import (
	"errors"
	"strings"

	"go-user-system/dao"
	"go-user-system/global"
	"go-user-system/model"
	"go-user-system/request"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrUserNotFound          = errors.New("username not found")
	ErrUserDisabled          = errors.New("user disabled")
	ErrPasswordWrong         = errors.New("password incorrect")
	ErrInvalidUserID         = errors.New("invalid user id")
	ErrNicknameTooLong       = errors.New("nickname too long")
	ErrNicknameEmpty         = errors.New("nickname is empty")
)

func Register(req request.RegisterRequest) error {

	username := strings.TrimSpace(req.Username)

	userInfo, err := dao.GetUserByUsername(global.DB, username)
	if err == nil && userInfo != nil {
		return ErrUsernameAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
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

func Login(req request.LoginRequest) (*model.User, error) {
	username := strings.TrimSpace(req.Username)

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

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrPasswordWrong
	}

	return user, nil
}

func GetProfile(userID int64) (*model.User, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}

	user, err := dao.GetUserByID(global.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if user.Status != model.UserStatusActive {
		return nil, ErrUserDisabled
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
