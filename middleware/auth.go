package middleware

import (
	"go-user-system/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Fail(c, http.StatusUnauthorized, 3001, "authorization header is empty")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, "", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Fail(c, http.StatusUnauthorized, 3002, "invalid authorization header")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			utils.Fail(c, http.StatusUnauthorized, 3003, "invalid token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserId)
		c.Set("username", claims.Username)
		c.Next()
	}
}
