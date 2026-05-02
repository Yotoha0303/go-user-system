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

type UpdateProfileRequest struct {
	Nickname string `json:"nickname"`
}

func RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, 1001, "参数错误")
		return
	}

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
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, 1001, "参数错误")
		return
	}

	user, err := service.Login(req.Username, req.Password)
	if err != nil {

		switch {
		case errors.Is(err, service.ErrUsernameEmpty),
			errors.Is(err, service.ErrPasswordEmpty):
			utils.Fail(c, 400, 1001, err.Error())
		case errors.Is(err, service.ErrUserNotFound),
			errors.Is(err, service.ErrPasswordWrong):
			utils.Fail(c, 400, 1004, "username or password incorrect")
		case errors.Is(err, service.ErrUserDisabled):
			utils.Fail(c, 403, 1005, err.Error())
		default:
			utils.Fail(c, 500, 1006, "login failed")
		}
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username)
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
	v, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, 401, 3004, "user context missing")
		return
	}

	userID, ok := v.(int64)
	if !ok {
		utils.Fail(c, 500, 3005, "invalid user context")
		return
	}

	user, err := service.GetProfile(userID)
	if err != nil {
		utils.Fail(c, 500, 3006, "get user profile failed")
		return
	}

	utils.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"nickname": user.Nickname,
		"status":   user.Status,
	})
}

func UpdateProfileHandler(c *gin.Context) {
	value, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, 400, 5001, "user_id not found in context")
		return
	}

	userID, ok := value.(int64)
	if !ok {
		utils.Fail(c, 500, 5002, "invalid user_id type")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, 5003, "参数错误")
		return
	}

	if err := service.UpdateNickname(userID, req.Nickname); err != nil {
		switch {
		case errors.Is(err, service.ErrNicknameEmpty),
			errors.Is(err, service.ErrNicknameTooLong):
			utils.Fail(c, 400, 5004, err.Error())
		case errors.Is(err, service.ErrUserNotFound),
			errors.Is(err, service.ErrUserDisabled):
			utils.Fail(c, 400, 5005, err.Error())
		default:
			utils.Fail(c, 500, 5006, "update profile failed")
		}
		return
	}
	utils.Success(c, nil)
}
