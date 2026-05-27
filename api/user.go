package api

import (
	"errors"
	"go-user-system/request"
	"go-user-system/response"
	"go-user-system/service"
	"go-user-system/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterHandler(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 1001, "参数错误")
		return
	}

	if err := service.Register(req.Username, req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameEmpty),
			errors.Is(err, service.ErrPasswordEmpty),
			errors.Is(err, service.ErrUsernameTooShort),
			errors.Is(err, service.ErrPasswordTooShort):
			response.Fail(c, http.StatusBadRequest, 1001, err.Error())
		case errors.Is(err, service.ErrUsernameExists):
			response.Fail(c, http.StatusConflict, 1002, err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, 1003, "注册失败")
		}
		return
	}

	response.Success(c, nil)
}

func LoginHandler(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 1001, "参数错误")
		return
	}

	user, err := service.Login(req.Username, req.Password)
	if err != nil {

		switch {
		case errors.Is(err, service.ErrUsernameEmpty),
			errors.Is(err, service.ErrPasswordEmpty):
			response.Fail(c, http.StatusBadRequest, 1001, err.Error())
		case errors.Is(err, service.ErrUserNotFound),
			errors.Is(err, service.ErrPasswordWrong):
			response.Fail(c, http.StatusNotFound, 1004, err.Error())
		case errors.Is(err, service.ErrUserDisabled):
			response.Fail(c, http.StatusConflict, 1005, err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, 1006, "登录错误")
		}
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 2005, "生成 access_token 失败")
		return
	}

	response.Success(c, gin.H{
		"access_token": token,
		"user": response.UserInfoResponse{
			ID:       user.ID,
			UserName: user.Username,
			NickName: user.Nickname,
			Status:   user.Status,
		},
	})
}

func MeHandler(c *gin.Context) {
	v, exists := c.Get("user_id")
	if !exists {
		response.Fail(c, http.StatusInternalServerError, 3004, "没有找到用户信息")
		return
	}

	userID, ok := v.(int64)
	if !ok {
		response.Fail(c, http.StatusInternalServerError, 3005, "无效的用户信息")
		return
	}

	user, err := service.GetProfile(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Fail(c, http.StatusNotFound, 3006, err.Error())
		case errors.Is(err, service.ErrUserDisabled):
			response.Fail(c, http.StatusConflict, 3007, err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, 3008, "获取用户信息失败")
		}
		return
	}

	response.Success(c, response.UserInfoResponse{
		ID:       user.ID,
		UserName: user.Username,
		NickName: user.Nickname,
		Status:   user.Status,
	})
}

func UpdateProfileHandler(c *gin.Context) {
	value, exists := c.Get("user_id")
	if !exists {
		response.Fail(c, http.StatusInternalServerError, 5001, "没有找到用户信息")
		return
	}

	userID, ok := value.(int64)
	if !ok {
		response.Fail(c, http.StatusInternalServerError, 5002, "无效的用户信息")
		return
	}

	var req request.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 5003, "参数错误")
		return
	}

	if err := service.UpdateNickname(userID, req.Nickname); err != nil {
		switch {
		case errors.Is(err, service.ErrNicknameEmpty),
			errors.Is(err, service.ErrNicknameTooLong):
			response.Fail(c, http.StatusBadRequest, 5004, err.Error())
		case errors.Is(err, service.ErrUserDisabled):
			response.Fail(c, http.StatusConflict, 5005, err.Error())
		case errors.Is(err, service.ErrUserNotFound):
			response.Fail(c, http.StatusNotFound, 5006, err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, 5007, "更改昵称失败")
		}
		return
	}
	response.Success(c, nil)
}
