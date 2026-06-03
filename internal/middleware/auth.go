package middleware

import (
	"errors"
	"go-user-system/internal/response"
	"go-user-system/internal/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			response.Fail(c, http.StatusUnauthorized, response.CodeTokenMissing, "authorization header is empty")
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.Fail(c, http.StatusUnauthorized, response.CodeTokenInvalidFormat, "invalid authorization Bearer")
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenString == "" {
			response.Fail(c, http.StatusUnauthorized, response.CodeTokenInvalidFormat, "invalid authorization header")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			switch {
			case errors.Is(err, jwt.ErrTokenMalformed):
				response.Fail(c, http.StatusUnauthorized, response.CodeTokenMalformed, "token is malformed")
			case errors.Is(err, jwt.ErrTokenSignatureInvalid):
				response.Fail(c, http.StatusUnauthorized, response.CodeTokenSignatureInvalid, "token signature is invalid")
			case errors.Is(err, jwt.ErrTokenExpired):
				response.Fail(c, http.StatusUnauthorized, response.CodeTokenExpired, "token is expired")
			default:
				response.Fail(c, http.StatusUnauthorized, response.CodeTokenInvalid, "invalid token")
			}
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
