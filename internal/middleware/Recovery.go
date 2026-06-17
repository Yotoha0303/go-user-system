package middleware

import (
	"go-user-system/internal/response"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID := GetRequestID(c)

				logger.Error(
					"request panic",
					slog.String("request_id", requestID),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.Any("panic", recovered),
					slog.String("stack", string(debug.Stack())),
				)

				if !c.Writer.Written() {
					response.Fail(
						c,
						http.StatusInternalServerError,
						response.CodeInternalError,
						"服务器内部错误",
					)
				}
				c.Abort()
			}

			if logger == nil {
				logger = slog.Default()
			}
		}()
		c.Next()
	}
}
