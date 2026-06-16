package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"
)

type requestIDContextKey struct{}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)

		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(RequestIDKey, requestID)

		ctx := context.WithValue(
			c.Request.Context(),
			requestIDContextKey{},
			requestID,
		)
		c.Request = c.Request.WithContext(ctx)

		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	value, exists := c.Get(RequestIDKey)
	if !exists {
		return ""
	}

	requestID, _ := value.(string)
	return requestID
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}
