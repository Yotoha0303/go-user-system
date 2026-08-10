package service

import (
	"errors"
	"go-user-system/internal/apperror"
	"go-user-system/internal/response"
	"net/http"
)

var (
	ErrAdminAlreadyBootstrapped = errors.New("administrator has already been bootstrapped")
	ErrRBACNotInitialized       = errors.New("RBAC roles are not initialized; run database migrations first")

	ErrUsernameTooShort = apperror.New(
		http.StatusBadRequest,
		response.CodeInvalidParams,
		"username too short",
	)

	ErrPasswordTooShortOrTooLong = apperror.New(
		http.StatusBadRequest,
		response.CodeInvalidParams,
		"password must contain at least 12 characters and no more than 72 UTF-8 bytes",
	)

	ErrUsernameAlreadyExists = apperror.New(
		http.StatusConflict,
		response.CodeUsernameAlreadyExists,
		"username already exists",
	)

	ErrUserNotFound = apperror.New(
		http.StatusNotFound,
		response.CodeUserNotFound,
		"username not found",
	)

	ErrUserPasswordNoDifference = apperror.New(
		http.StatusConflict,
		response.CodeUserPasswordNoDifference,
		"user password no difference",
	)

	ErrUserEnteredTheOldPasswordIncorrectly = apperror.New(
		http.StatusConflict,
		response.CodeUserPasswordNoDifference,
		"user entered the old password incorrectly",
	)

	ErrUserDisabled = apperror.New(
		http.StatusForbidden,
		response.CodeUserDisabled,
		"user disabled",
	)

	ErrInvalidCredentials = apperror.New(
		http.StatusUnauthorized,
		response.CodeLoginFailed,
		"username or password incorrect",
	)

	ErrInvalidUserID = apperror.New(
		http.StatusBadRequest,
		response.CodeInvalidParams,
		"invalid user id",
	)

	ErrNicknameTooLong = apperror.New(
		http.StatusBadRequest,
		response.CodeNicknameInvalid,
		"nickname too long",
	)

	ErrNicknameEmpty = apperror.New(
		http.StatusBadRequest,
		response.CodeNicknameInvalid,
		"nickname is empty",
	)

	ErrDatabaseNotInitialized = apperror.New(
		http.StatusInternalServerError,
		response.CodeDatabaseNotInitialized,
		"database is not initialized",
	)

	ErrRefreshTokenInvalid = apperror.New(
		http.StatusUnauthorized,
		response.CodeRefreshTokenInvalid,
		"refresh token is invalid",
	)

	ErrRefreshTokenExpired = apperror.New(
		http.StatusUnauthorized,
		response.CodeRefreshTokenExpired,
		"refresh token is expired",
	)

	ErrRefreshTokenRevoked = apperror.New(
		http.StatusUnauthorized,
		response.CodeRefreshTokenRevoked,
		"refresh token has been revoked",
	)

	ErrRefreshTokenReplay = apperror.New(
		http.StatusUnauthorized,
		response.CodeRefreshTokenRevoked,
		"refresh token replay detected",
	)

	ErrAccessSessionInvalid = apperror.New(
		http.StatusUnauthorized,
		response.CodeTokenInvalid,
		"access session is invalid",
	)

	ErrLoginRateLimited = apperror.New(
		http.StatusTooManyRequests,
		response.CodeLoginRateLimited,
		"too many login attempts",
	)

	ErrPermissionDenied = apperror.New(
		http.StatusForbidden,
		response.CodePermissionDenied,
		"permission denied",
	)

	ErrRoleNotFound = apperror.New(
		http.StatusNotFound,
		response.CodeRoleNotFound,
		"role not found",
	)
)
