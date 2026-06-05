package handler

import (
	"go-user-system/internal/apperror"
	"go-user-system/internal/model"
	"go-user-system/internal/request"
	"go-user-system/internal/response"
	"go-user-system/internal/service"
	"go-user-system/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserService interface {
	Register(req request.RegisterRequest) error
	Login(req request.LoginRequest) (*model.User, error)
	GetProfile(userID int64) (*model.User, error)
	UpdateNickname(userID int64, nickname string) error
}

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

var _ UserService = (*service.UserService)(nil)

func (h *UserHandler) RegisterHandler(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}

	if err := h.userService.Register(req); err != nil {
		handleError(c, err, response.CodeRegisterFailed, "注册失败")
		return
	}

	response.Success(c, nil)
}

func (h *UserHandler) LoginHandler(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}

	user, err := h.userService.Login(req)
	if err != nil {
		handleError(c, err, response.CodeLoginFailed, "登录错误")
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username)
	if err != nil {
		handleError(
			c,
			apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeTokenGenerateFailed,
				"生成 access_token 失败",
				err,
			),
			response.CodeTokenGenerateFailed,
			"生成 access_token 失败",
		)
		return
	}

	response.Success(c, response.TokenAndUserInfoResponse{
		AccessToken: token,
		User: response.UserInfoResponse{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Status:   user.Status,
		},
	})
}

func (h *UserHandler) MeHandler(c *gin.Context) {
	value, exists := c.Get("user_id")
	if !exists {
		handleError(
			c,
			apperror.New(
				http.StatusInternalServerError,
				response.CodeTokenUserMissing,
				"没有找到用户信息",
			),
			response.CodeGetProfileFailed,
			"获取用户信息失败",
		)
		return
	}

	userID, ok := value.(int64)
	if !ok {
		handleError(
			c,
			apperror.New(
				http.StatusInternalServerError,
				response.CodeTokenUserInvalid,
				"无效的用户信息",
			),
			response.CodeGetProfileFailed,
			"获取用户信息失败",
		)
		return
	}

	user, err := h.userService.GetProfile(userID)
	if err != nil {
		handleError(c, err, response.CodeGetProfileFailed, "获取用户信息失败")
		return
	}

	response.Success(c, response.UserInfoResponse{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		Status:   user.Status,
	})
}

func (h *UserHandler) UpdateProfileHandler(c *gin.Context) {
	value, exists := c.Get("user_id")
	if !exists {
		handleError(
			c,
			apperror.New(
				http.StatusInternalServerError,
				response.CodeTokenUserMissing,
				"没有找到用户信息",
			),
			response.CodeUpdateNicknameFailed,
			"更改昵称失败",
		)
		return
	}

	userID, ok := value.(int64)
	if !ok {
		handleError(
			c,
			apperror.New(
				http.StatusInternalServerError,
				response.CodeTokenUserInvalid,
				"无效的用户信息",
			),
			response.CodeUpdateNicknameFailed,
			"更改昵称失败",
		)
		return
	}

	var req request.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}

	if err := h.userService.UpdateNickname(userID, req.Nickname); err != nil {
		handleError(c, err, response.CodeUpdateNicknameFailed, "更改昵称失败")
		return
	}

	response.Success(c, nil)
}
