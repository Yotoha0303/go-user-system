package handler

import (
	"context"
	"errors"
	"fmt"
	"go-user-system/internal/apperror"
	"go-user-system/internal/auth"
	"go-user-system/internal/model"
	"go-user-system/internal/request"
	"go-user-system/internal/response"
	"go-user-system/internal/service"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	refreshTokenCookieName = "refresh_token"
	refreshTokenCookiePath = "/api/v1/auth" // #nosec G101
)

type UserService interface {
	Register(ctx context.Context, req request.RegisterRequest) error
	Login(ctx context.Context, req request.LoginRequest) (*model.User, error)
	GetProfile(ctx context.Context, userID int64) (*model.User, error)
	UpdateNickname(ctx context.Context, userID int64, nickname string) error
	UpdateUserPassword(ctx context.Context, userID int64, req request.UpdatePasswordRequest) error
}

type AuthSessionService interface {
	StoreRefreshToken(ctx context.Context, token *model.RefreshToken) error
	RotateRefreshToken(ctx context.Context, userID int64, authVersion int64, oldJTI string, oldHash string, next *model.RefreshToken) error
	RevokeRefreshToken(ctx context.Context, userID int64, jti string, tokenHash string) error
	RevokeAccessToken(ctx context.Context, jti string, expiresAt time.Time) error
	CheckLoginAllowed(ctx context.Context, account, ip string) (time.Duration, error)
	RecordLoginFailure(ctx context.Context, account, ip string) (time.Duration, error)
	ResetLoginFailures(ctx context.Context, account string) error
}

type UserHandler struct {
	userService          UserService
	authSessionService   AuthSessionService
	generateToken        func(userID int64, username string, authVersion int64) (string, error)
	generateRefreshToken func(userID int64, username string, authVersion int64) (*auth.IssuedToken, error)
	parseAccessToken     func(tokenString string) (*auth.UserClaims, error)
	parseRefreshToken    func(tokenString string) (*auth.UserClaims, error)
	hashToken            func(token string) string
	accessTokenExpiresIn func() int64
	secureCookies        bool
}

type UserHandlerOptions struct {
	SecureCookies bool
}

func NewUserHandler(userService UserService, tokenManager *auth.TokenManager, authSessionService ...AuthSessionService) *UserHandler {
	return NewUserHandlerWithOptions(userService, tokenManager, UserHandlerOptions{}, authSessionService...)
}

func NewUserHandlerWithOptions(userService UserService, tokenManager *auth.TokenManager, options UserHandlerOptions, authSessionService ...AuthSessionService) *UserHandler {
	var sessionService AuthSessionService
	if len(authSessionService) > 0 {
		sessionService = authSessionService[0]
	}

	return &UserHandler{
		userService:          userService,
		authSessionService:   sessionService,
		generateToken:        tokenManager.GenerateAccessToken,
		generateRefreshToken: tokenManager.GenerateRefreshToken,
		parseAccessToken:     tokenManager.ParseAccessToken,
		parseRefreshToken:    tokenManager.ParseRefreshToken,
		hashToken:            auth.HashToken,
		secureCookies:        options.SecureCookies,
		accessTokenExpiresIn: func() int64 {
			return int64(tokenManager.AccessTokenTTL().Seconds())
		},
	}
}

var _ UserService = (*service.UserService)(nil)

// RegisterHandler godoc
// @Summary 用户注册
// @Tags auth
// @Accept json
// @Produce json
// @Param body body request.RegisterRequest true "注册参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/auth/register [post]
func (h *UserHandler) RegisterHandler(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}

	if err := h.userService.Register(c.Request.Context(), req); err != nil {
		handleError(c, err, response.CodeRegisterFailed, "register failed")
		return
	}

	response.Success(c, nil)
}

// LoginHandler godoc
// @Summary 用户登录
// @Tags auth
// @Accept json
// @Produce json
// @Param body body request.LoginRequest true "登录参数"
// @Success 200 {object} response.Response{data=response.TokenAndUserInfoResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 429 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/auth/login [post]
func (h *UserHandler) LoginHandler(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}

	loginIP := strings.TrimSpace(c.ClientIP())
	if loginIP == "" {
		loginIP = "unknown"
	}
	if h.authSessionService != nil {
		retryAfter, err := h.authSessionService.CheckLoginAllowed(c.Request.Context(), req.Username, loginIP)
		if err != nil {
			handleError(c, err, response.CodeAuthStateUnavailable, "登录限流检查失败")
			return
		}
		if respondLoginRateLimited(c, retryAfter) {
			return
		}
	}

	user, err := h.userService.Login(c.Request.Context(), req)
	if err != nil {
		if h.authSessionService != nil && errors.Is(err, service.ErrInvalidCredentials) {
			retryAfter, rateErr := h.authSessionService.RecordLoginFailure(c.Request.Context(), req.Username, loginIP)
			if rateErr != nil {
				handleError(c, rateErr, response.CodeAuthStateUnavailable, "记录登录失败次数失败")
				return
			}
			if respondLoginRateLimited(c, retryAfter) {
				return
			}
		}
		handleError(c, err, response.CodeLoginFailed, "登录错误")
		return
	}
	if h.authSessionService != nil {
		if err := h.authSessionService.ResetLoginFailures(c.Request.Context(), req.Username); err != nil {
			handleError(c, err, response.CodeAuthStateUnavailable, "清理登录失败次数失败")
			return
		}
	}

	token, err := h.generateToken(user.ID, user.Username, user.AuthVersion)
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

	refreshToken, err := h.generateRefreshToken(user.ID, user.Username, user.AuthVersion)
	if err != nil {
		handleError(
			c,
			apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeTokenGenerateFailed,
				"生成 refresh_token 失败",
				err,
			),
			response.CodeTokenGenerateFailed,
			"生成 refresh_token 失败",
		)
		return
	}

	if h.authSessionService != nil {
		if err := h.authSessionService.StoreRefreshToken(
			c.Request.Context(),
			refreshTokenModel(user.ID, refreshToken, h.hashToken(refreshToken.Token)),
		); err != nil {
			handleError(c, err, response.CodeRefreshTokenInvalid, "保存 refresh_token 失败")
			return
		}
	}

	h.setRefreshTokenCookie(c, refreshToken)
	response.Success(c, response.TokenAndUserInfoResponse{
		AccessToken:           token,
		AccessTokenExpiresIn:  h.accessTokenExpiresIn(),
		RefreshTokenExpiresIn: refreshToken.ExpiresIn,
		User: response.UserInfoResponse{
			ID:          user.ID,
			Username:    user.Username,
			Nickname:    user.Nickname,
			Status:      user.Status,
			LastLoginAt: user.LastLoginAt,
		},
	})
}

// RefreshTokenHandler godoc
// @Summary 刷新双 Token
// @Tags auth
// @Accept json
// @Produce json
// @Description 浏览器客户端优先通过 HttpOnly Cookie 发送 Refresh Token，请求体仅用于非浏览器客户端兼容。
// @Param body body request.RefreshTokenRequest false "Refresh Token"
// @Success 200 {object} response.Response{data=response.TokenPairResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/auth/refresh [post]
func (h *UserHandler) RefreshTokenHandler(c *gin.Context) {
	var req request.RefreshTokenRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	refreshTokenValue := requestRefreshToken(c, req.RefreshToken)
	if refreshTokenValue == "" {
		h.clearRefreshTokenCookie(c)
		response.Fail(c, http.StatusUnauthorized, response.CodeRefreshTokenInvalid, "refresh token is missing")
		return
	}

	claims, err := h.parseRefreshToken(refreshTokenValue)
	if err != nil {
		h.clearRefreshTokenCookie(c)
		handleRefreshTokenParseError(c, err)
		return
	}

	accessToken, err := h.generateToken(claims.UserID, claims.Username, claims.AuthVersion)
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

	nextRefreshToken, err := h.generateRefreshToken(claims.UserID, claims.Username, claims.AuthVersion)
	if err != nil {
		handleError(
			c,
			apperror.Wrap(
				http.StatusInternalServerError,
				response.CodeTokenGenerateFailed,
				"生成 refresh_token 失败",
				err,
			),
			response.CodeTokenGenerateFailed,
			"生成 refresh_token 失败",
		)
		return
	}

	if h.authSessionService == nil {
		handleError(
			c,
			apperror.New(
				http.StatusInternalServerError,
				response.CodeDatabaseNotInitialized,
				"refresh token service is not initialized",
			),
			response.CodeRefreshTokenInvalid,
			"刷新 token 失败",
		)
		return
	}

	if err := h.authSessionService.RotateRefreshToken(
		c.Request.Context(),
		claims.UserID,
		claims.AuthVersion,
		claims.JTI,
		h.hashToken(refreshTokenValue),
		refreshTokenModel(claims.UserID, nextRefreshToken, h.hashToken(nextRefreshToken.Token)),
	); err != nil {
		handleError(c, err, response.CodeRefreshTokenInvalid, "刷新 token 失败")
		return
	}

	h.setRefreshTokenCookie(c, nextRefreshToken)
	response.Success(c, response.TokenPairResponse{
		AccessToken:           accessToken,
		AccessTokenExpiresIn:  h.accessTokenExpiresIn(),
		RefreshTokenExpiresIn: nextRefreshToken.ExpiresIn,
	})
}

// LogoutHandler godoc
// @Summary 退出登录并吊销 Refresh Token
// @Tags auth
// @Accept json
// @Produce json
// @Description 浏览器客户端优先通过 HttpOnly Cookie 发送 Refresh Token，请求体仅用于非浏览器客户端兼容。
// @Param body body request.LogoutRequest false "Refresh Token"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/auth/logout [post]
func (h *UserHandler) LogoutHandler(c *gin.Context) {
	var req request.LogoutRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	refreshTokenValue := requestRefreshToken(c, req.RefreshToken)
	if refreshTokenValue == "" {
		h.clearRefreshTokenCookie(c)
		response.Fail(c, http.StatusUnauthorized, response.CodeRefreshTokenInvalid, "refresh token is missing")
		return
	}

	claims, err := h.parseRefreshToken(refreshTokenValue)
	if err != nil {
		h.clearRefreshTokenCookie(c)
		handleRefreshTokenParseError(c, err)
		return
	}

	if h.authSessionService == nil {
		handleError(
			c,
			apperror.New(
				http.StatusInternalServerError,
				response.CodeDatabaseNotInitialized,
				"refresh token service is not initialized",
			),
			response.CodeRefreshTokenInvalid,
			"退出登录失败",
		)
		return
	}

	if accessClaims, ok := h.optionalAccessToken(c); ok {
		if accessClaims.UserID != claims.UserID {
			response.Fail(c, http.StatusUnauthorized, response.CodeTokenInvalid, "access token does not match refresh token")
			return
		}
		if err := h.authSessionService.RevokeAccessToken(c.Request.Context(), accessClaims.JTI, accessClaims.ExpiresAt.Time); err != nil {
			handleError(c, err, response.CodeAuthStateUnavailable, "吊销 access token 失败")
			return
		}
	}

	if err := h.authSessionService.RevokeRefreshToken(
		c.Request.Context(),
		claims.UserID,
		claims.JTI,
		h.hashToken(refreshTokenValue),
	); err != nil {
		handleError(c, err, response.CodeRefreshTokenInvalid, "退出登录失败")
		return
	}

	h.clearRefreshTokenCookie(c)
	response.Success(c, nil)
}

// MeHandler godoc
// @Summary 获取当前用户资料
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=response.UserInfoResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users/me [get]
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

// UpdateProfileHandler godoc
// @Summary 修改当前用户昵称
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body request.UpdateProfileRequest true "昵称参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users/me/profile [put]
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

func refreshTokenModel(userID int64, issuedToken *auth.IssuedToken, tokenHash string) *model.RefreshToken {
	return &model.RefreshToken{
		UserID:    userID,
		JTI:       issuedToken.JTI,
		FamilyID:  issuedToken.JTI,
		TokenHash: tokenHash,
		ExpiresAt: issuedToken.ExpiresAt,
	}
}

func bindOptionalJSON(c *gin.Context, target interface{}) error {
	err := c.ShouldBindJSON(target)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func requestRefreshToken(c *gin.Context, bodyToken string) string {
	if token := strings.TrimSpace(bodyToken); token != "" {
		return token
	}
	token, err := c.Cookie(refreshTokenCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(token)
}

func (h *UserHandler) setRefreshTokenCookie(c *gin.Context, issuedToken *auth.IssuedToken) {
	if issuedToken == nil {
		return
	}
	//nolint:gosec
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    issuedToken.Token,
		Path:     refreshTokenCookiePath,
		Expires:  issuedToken.ExpiresAt,
		MaxAge:   int(issuedToken.ExpiresIn),
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *UserHandler) clearRefreshTokenCookie(c *gin.Context) {
	//nolint:gosec
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     refreshTokenCookiePath,
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func respondLoginRateLimited(c *gin.Context, retryAfter time.Duration) bool {
	if retryAfter <= 0 {
		return false
	}
	seconds := int64((retryAfter + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	c.Header("Retry-After", fmt.Sprintf("%d", seconds))
	handleError(c, service.ErrLoginRateLimited, response.CodeLoginRateLimited, "登录尝试过于频繁")
	return true
}

func (h *UserHandler) optionalAccessToken(c *gin.Context) (*auth.UserClaims, bool) {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader == "" {
		return nil, false
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, false
	}
	tokenValue := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if tokenValue == "" {
		return nil, false
	}
	claims, err := h.parseAccessToken(tokenValue)
	if err != nil {
		return nil, false
	}
	return claims, true
}

func handleRefreshTokenParseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		response.Fail(c, http.StatusUnauthorized, response.CodeRefreshTokenExpired, "refresh token is expired")
	case errors.Is(err, jwt.ErrTokenMalformed):
		response.Fail(c, http.StatusUnauthorized, response.CodeRefreshTokenInvalid, "refresh token is malformed")
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		response.Fail(c, http.StatusUnauthorized, response.CodeRefreshTokenInvalid, "refresh token signature is invalid")
	default:
		response.Fail(c, http.StatusUnauthorized, response.CodeRefreshTokenInvalid, "refresh token is invalid")
	}
}

// UpdateUserPasswordHandler godoc
// @Summary 修改当前用户密码
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body request.UpdatePasswordRequest true "密码参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users/me/update/password [patch]
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
			response.CodeTokenUserMissing,
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
			response.CodeTokenUserInvalid,
			"更改昵称失败",
		)
		return
	}

	var req request.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}

	if err := h.userService.UpdateUserPassword(c.Request.Context(), userID, req); err != nil {
		handleError(c, err, response.CodeUpdateUserPasswordFailed, "修改密码失败")
		return
	}

	h.clearRefreshTokenCookie(c)
	response.Success(c, nil)
}
