package handler

import (
	"go-user-system/internal/apperror"
	"go-user-system/internal/response"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	RequestIDHeader = "X-Request-ID"
)

func handleError(c *gin.Context, err error, fallbackCode int, fallbackMessage string) {
	logger := slog.Default()
	requestID := c.GetHeader(RequestIDHeader)

	if appErr, ok := apperror.FromError(err); ok {
		if appErr.Cause != nil {

			attrs := []slog.Attr{
				slog.String("request_id", requestID),
				slog.String("path", c.Request.URL.Path),
				slog.String("method", c.Request.Method),
				slog.Int("code", appErr.Code),
				slog.String("message", appErr.Message),
				slog.Any("cause", appErr.Error()),
			}

			logger.LogAttrs(c.Request.Context(), slog.LevelError, "app error", attrs...)
		}
		response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
		return
	}

	attrs := []slog.Attr{
		slog.String("request_id", requestID),
		slog.String("path", c.Request.URL.Path),
		slog.String("method", c.Request.Method),
		slog.Any("cause", err),
	}

	logger.LogAttrs(c.Request.Context(), slog.LevelError, "app error", attrs...)

	response.Fail(c, http.StatusInternalServerError, fallbackCode, fallbackMessage)
}
