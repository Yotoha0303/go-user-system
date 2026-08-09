package middleware

import (
	"context"
	"errors"
	"go-user-system/internal/apperror"
	"go-user-system/internal/auth"
	"go-user-system/internal/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AccessTokenValidator interface {
	ValidateAccessToken(ctx context.Context, claims *auth.UserClaims) error
}

func AuthMiddleware(tokenManager *auth.TokenManager, validators ...AccessTokenValidator) gin.HandlerFunc {
	var validator AccessTokenValidator
	if len(validators) > 0 {
		validator = validators[0]
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if strings.TrimSpace(authHeader) == "" {
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

		claims, err := tokenManager.ParseAccessToken(tokenString)
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

		if validator != nil {
			if err := validator.ValidateAccessToken(c.Request.Context(), claims); err != nil {
				if appErr, ok := apperror.FromError(err); ok && appErr.HTTPStatus == http.StatusServiceUnavailable {
					response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
				} else {
					response.Fail(c, http.StatusUnauthorized, response.CodeTokenInvalid, "invalid access session")
				}
				c.Abort()
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("access_token", tokenString)
		c.Set("access_jti", claims.JTI)
		c.Next()
	}
}
