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

var (
	CodeInvalidParams         = 1001
	CodeUsernameAlreadyExists = 1002
	CodeRegisterFailed        = 1003
	CodeUserNotFound          = 1004
	CodeUserDisabled          = 1005
	CodeLoginFailed           = 1006

	CodeTokenGenerateFailed = 2001
	CodeTokenUserMissing    = 2002
	CodeTokenUserInvalid    = 2003
	CodeGetProfileFailed    = 2004

	CodeNicknameInvalid      = 3001
	CodeUpdateNicknameFailed = 3002
)

func RegisterHandler(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, CodeInvalidParams, "参数错误")
		return
	}

	if err := service.Register(req); err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameAlreadyExists):
			response.Fail(c, http.StatusConflict, CodeUsernameAlreadyExists, err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, CodeRegisterFailed, "注册失败")
		}
		return
	}

	response.Success(c, nil)
}

func LoginHandler(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, CodeInvalidParams, "参数错误")
		return
	}

	user, err := service.Login(req)
	if err != nil {

		switch {
		case errors.Is(err, service.ErrUserNotFound),
			errors.Is(err, service.ErrPasswordWrong):
			response.Fail(c, http.StatusUnauthorized, CodeUserNotFound, "username or password incorrect")
		case errors.Is(err, service.ErrUserDisabled):
			response.Fail(c, http.StatusForbidden, CodeUserDisabled, err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, CodeLoginFailed, "登录错误")
		}
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, CodeTokenGenerateFailed, "生成 access_token 失败")
		return
	}

	response.Success(c, gin.H{
		"access_token": token,
		"user": response.UserInfoResponse{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Status:   user.Status,
		},
	})
}

func MeHandler(c *gin.Context) {
	v, exists := c.Get("user_id")
	if !exists {
		response.Fail(c, http.StatusInternalServerError, CodeTokenUserMissing, "没有找到用户信息")
		return
	}

	userID, ok := v.(int64)
	if !ok {
		response.Fail(c, http.StatusInternalServerError, CodeTokenUserInvalid, "无效的用户信息")
		return
	}

	user, err := service.GetProfile(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Fail(c, http.StatusNotFound, CodeUserNotFound, err.Error())
		case errors.Is(err, service.ErrUserDisabled):
			response.Fail(c, http.StatusForbidden, CodeUserDisabled, err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, CodeGetProfileFailed, "获取用户信息失败")
		}
		return
	}

	response.Success(c, response.UserInfoResponse{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		Status:   user.Status,
	})
}

func UpdateProfileHandler(c *gin.Context) {
	value, exists := c.Get("user_id")
	if !exists {
		response.Fail(c, http.StatusInternalServerError, CodeTokenUserMissing, "没有找到用户信息")
		return
	}

	userID, ok := value.(int64)
	if !ok {
		response.Fail(c, http.StatusInternalServerError, CodeTokenUserInvalid, "无效的用户信息")
		return
	}

	var req request.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, CodeInvalidParams, "参数错误")
		return
	}

	if err := service.UpdateNickname(userID, req.Nickname); err != nil {
		switch {
		case errors.Is(err, service.ErrNicknameEmpty), errors.Is(err, service.ErrNicknameTooLong):
			response.Fail(c, http.StatusBadRequest, CodeNicknameInvalid, err.Error())
		case errors.Is(err, service.ErrUserDisabled):
			response.Fail(c, http.StatusForbidden, CodeUserDisabled, err.Error())
		case errors.Is(err, service.ErrUserNotFound):
			response.Fail(c, http.StatusNotFound, CodeUserNotFound, err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, CodeUpdateNicknameFailed, "更改昵称失败")
		}
		return
	}
	response.Success(c, nil)
}
