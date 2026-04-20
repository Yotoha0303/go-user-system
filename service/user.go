package service

import (
	"go-user-system/dao"
	"go-user-system/global"
	"go-user-system/model"
)

func Register(username, password string) error {
	user := model.User{
		Username: username,
		Password: password,
	}
	return dao.CreateUser(global.DB, &user)
}
