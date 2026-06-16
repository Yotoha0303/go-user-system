package handler

import (
	"context"
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
	Register(ctx context.Context, req request.RegisterRequest) error
	Login(ctx context.Context, req request.LoginRequest) (*model.User, error)
	GetProfile(ctx context.Context, userID int64) (*model.User, error)
	UpdateNickname(ctx context.Context, userID int64, nickname string) error
	UpdateUserPassword(ctx context.Context, userID int64, req request.UpdatePasswordRequest) error
}

type UserHandler struct {
	userService   UserService
	generateToken func(userID int64, username string) (string, error)
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		userService:   userService,
		generateToken: utils.GenerateToken,
	}
}

var _ UserService = (*service.UserService)(nil)

func (h *UserHandler) RegisterHandler(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}

	if err := h.userService.Register(c.Request.Context(), req); err != nil {
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

	user, err := h.userService.Login(c.Request.Context(), req)
	if err != nil {
		handleError(c, err, response.CodeLoginFailed, "登录错误")
		return
	}

	token, err := h.generateToken(user.ID, user.Username)
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
			ID:          user.ID,
			Username:    user.Username,
			Nickname:    user.Nickname,
			Status:      user.Status,
			LastLoginAt: user.LastLoginAt,
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

	user, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err, response.CodeGetProfileFailed, "获取用户信息失败")
		return
	}

	response.Success(c, response.UserInfoResponse{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Status:      user.Status,
		LastLoginAt: user.LastLoginAt,
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

	if err := h.userService.UpdateNickname(c.Request.Context(), userID, req.Nickname); err != nil {
		handleError(c, err, response.CodeUpdateNicknameFailed, "更改昵称失败")
		return
	}

	response.Success(c, nil)
}

func (h *UserHandler) UpdateUserPasswordHandler(c *gin.Context) {
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

	var req request.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}

	if err := h.userService.UpdateUserPassword(c, userID, req); err != nil {
		handleError(c, err, http.StatusInternalServerError, "修改密码失败")
		return
	}

	response.Success(c, nil)
}

func (h *UserHandler) LoginOutHandler(c *gin.Context) {
	response.Success(c, nil)
}
