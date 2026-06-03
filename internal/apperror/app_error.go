package apperror

import "errors"

type AppError struct {
	HTTPStatus int
	Code       int
	Message    string
	Cause      error
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	if e.Cause != nil {
		return e.Cause.Error()
	}

	return "application error"
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(httpStatus int, code int, message string) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
	}
}

func Wrap(httpStatus int, code int, message string, cause error) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
		Cause:      cause,
	}
}

func FromError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
