package middleware

import (
	"errors"
	"go-user-system/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			utils.Fail(c, http.StatusUnauthorized, 3001, "authorization header is empty")
			c.Abort()
			return
		}

		// parts := strings.SplitN(authHeader, "", 2)
		// if len(parts) != 2 || parts[0] != "Bearer" {
		// 	utils.Fail(c, http.StatusUnauthorized, 3002, "invalid authorization header")
		// 	c.Abort()
		// 	return
		// }

		// claims, err := utils.ParseToken(parts[1])
		// if err != nil {
		// 	utils.Fail(c, http.StatusUnauthorized, 3003, "invalid token")
		// 	c.Abort()
		// 	return
		// }
		if !strings.HasPrefix(authHeader, "Bearer ") {
			utils.Fail(c, http.StatusUnauthorized, 3002, "invalid authorization header")
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenString == "" {
			utils.Fail(c, http.StatusUnauthorized, 3002, "invalid authorization header")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			switch {
			case errors.Is(err, jwt.ErrTokenMalformed):
				utils.Fail(c, http.StatusUnauthorized, 3003, "token is malformed")
			case errors.Is(err, jwt.ErrTokenSignatureInvalid):
				utils.Fail(c, http.StatusUnauthorized, 3003, "token signature is invalid")
			case errors.Is(err, jwt.ErrTokenExpired):
				utils.Fail(c, http.StatusUnauthorized, 3003, "token is expired")
			default:
				utils.Fail(c, http.StatusUnauthorized, 3003, "invalid token")
			}
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
