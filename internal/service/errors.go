package service

import (
	"go-user-system/internal/apperror"
	"go-user-system/internal/response"
	"net/http"
)

var (
	ErrUsernameTooShort = apperror.New(
		http.StatusBadRequest,
		response.CodeInvalidParams,
		"username too short",
	)

	ErrPasswordTooShort = apperror.New(
		http.StatusBadRequest,
		response.CodeInvalidParams,
		"password too short",
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
)
