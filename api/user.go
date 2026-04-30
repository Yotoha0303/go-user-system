package api

import (
	"errors"
	"go-user-system/service"
	"go-user-system/utils"

	"github.com/gin-gonic/gin"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, 1001, "参数错误")
		return
	}

	// err := service.Register(req.Username, req.Password)
	// if err != nil {
	// 	utils.Fail(c, 500, 1002, "注册失败")
	// 	return
	// }

	if err := service.Register(req.Username, req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameEmpty),
			errors.Is(err, service.ErrPasswordEmpty),
			errors.Is(err, service.ErrUsernameTooShort),
			errors.Is(err, service.ErrPasswordTooShort):
			utils.Fail(c, 400, 1001, err.Error())
		case errors.Is(err, service.ErrUsernameExists):
			utils.Fail(c, 400, 1002, err.Error())
		default:
			utils.Fail(c, 500, 1003, "register failed")
		}
		return
	}

	utils.Success(c, nil)
}

func LoginHandler(c *gin.Context) {
	//1、前端会提供账号和密码，需要接受前端发送来的账户信息
	var req LoginRequest //此处不复用注册时的结构体
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, 1001, "参数错误")
		return
	}

	//2、service层处理login请求
	user, err := service.Login(req.Username, req.Password)
	if err != nil {
		//错误情况：账户不存在、密码错误、该账户已经禁用
		// switch err.Error() {
		// case "username is not exist", "password is failed", "this account is ben":
		// 	return
		// default:
		// 	utils.Fail(c, 500, 1003, "login exists unknow error")
		// 	return
		// }

		//改
		switch {
		case errors.Is(err, service.ErrUsernameEmpty),
			errors.Is(err, service.ErrPasswordEmpty):
			utils.Fail(c, 400, 1001, err.Error())
		case errors.Is(err, service.ErrUserNotFound),
			errors.Is(err, service.ErrPasswordWrong):
			utils.Fail(c, 400, 1004, "username or password incorrect")
		case errors.Is(err, service.ErrUserDisabled):
			utils.Fail(c, 403, 1005, "user disabled")
		default:
			utils.Fail(c, 500, 1006, "login failed")
		}
		return
	}

	token, err := utils.GenerateToken(uint(user.ID), user.Username)
	if err != nil {
		utils.Fail(c, 500, 2005, "generate token failed")
		return
	}

	utils.Success(c, gin.H{
		"access_token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"status":   user.Status,
		},
	})
}

func MeHandler(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	utils.Success(c, gin.H{
		"user_id":  userID,
		"username": username,
	})
}
