package middleware

import (
	"context"
	"go-user-system/internal/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type PermissionChecker interface {
	HasPermission(ctx context.Context, userID int64, permissionCode string) (bool, error)
}

func RequirePermission(checker PermissionChecker, permissionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if checker == nil || strings.TrimSpace(permissionCode) == "" {
			response.Fail(c, http.StatusInternalServerError, response.CodeRBACFailed, "权限配置错误")
			c.Abort()
			return
		}

		value, exists := c.Get("user_id")
		if !exists {
			response.Fail(c, http.StatusUnauthorized, response.CodeTokenUserMissing, "用户未认证")
			c.Abort()
			return
		}

		userID, ok := value.(int64)
		if !ok || userID <= 0 {
			response.Fail(c, http.StatusUnauthorized, response.CodeTokenUserInvalid, "无效的用户信息")
			c.Abort()
			return
		}

		allowed, err := checker.HasPermission(c.Request.Context(), userID, permissionCode)
		if err != nil {
			response.Fail(c, http.StatusInternalServerError, response.CodeRBACFailed, "权限校验失败")
			c.Abort()
			return
		}
		if !allowed {
			response.Fail(c, http.StatusForbidden, response.CodePermissionDenied, "无权限访问")
			c.Abort()
			return
		}

		c.Next()
	}
}
