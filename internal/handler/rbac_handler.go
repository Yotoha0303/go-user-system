package handler

import (
	"context"
	"go-user-system/internal/model"
	"go-user-system/internal/request"
	"go-user-system/internal/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RBACService interface {
	ListRoles(ctx context.Context) ([]model.Role, error)
	ListPermissions(ctx context.Context) ([]model.Permission, error)
	GetUserAuthorization(ctx context.Context, userID int64) ([]string, []string, error)
	AssignRolesToUser(ctx context.Context, userID int64, roleCodes []string) error
}

type RBACHandler struct {
	rbacService RBACService
}

func NewRBACHandler(rbacService RBACService) *RBACHandler {
	return &RBACHandler{rbacService: rbacService}
}

// ListRolesHandler godoc
// @Summary 查询角色列表
// @Tags admin-rbac
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]response.RoleResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/admin/roles [get]
func (h *RBACHandler) ListRolesHandler(c *gin.Context) {
	roles, err := h.rbacService.ListRoles(c.Request.Context())
	if err != nil {
		handleError(c, err, response.CodeRBACFailed, "读取角色列表失败")
		return
	}

	res := make([]response.RoleResponse, 0, len(roles))
	for _, role := range roles {
		res = append(res, response.RoleResponse{
			ID:   role.ID,
			Code: role.Code,
			Name: role.Name,
		})
	}
	response.Success(c, res)
}

// ListPermissionsHandler godoc
// @Summary 查询权限列表
// @Tags admin-rbac
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]response.PermissionResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/admin/permissions [get]
func (h *RBACHandler) ListPermissionsHandler(c *gin.Context) {
	permissions, err := h.rbacService.ListPermissions(c.Request.Context())
	if err != nil {
		handleError(c, err, response.CodeRBACFailed, "读取权限列表失败")
		return
	}

	res := make([]response.PermissionResponse, 0, len(permissions))
	for _, permission := range permissions {
		res = append(res, response.PermissionResponse{
			ID:     permission.ID,
			Code:   permission.Code,
			Name:   permission.Name,
			Method: permission.Method,
			Path:   permission.Path,
		})
	}
	response.Success(c, res)
}

// GetMyAuthorizationHandler godoc
// @Summary 查询当前用户角色和权限码
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=response.AuthorizationInfoResponse}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users/me/authorization [get]
func (h *RBACHandler) GetMyAuthorizationHandler(c *gin.Context) {
	value, exists := c.Get("user_id")
	if !exists {
		response.Fail(c, http.StatusUnauthorized, response.CodeTokenUserMissing, "用户未认证")
		return
	}

	userID, ok := value.(int64)
	if !ok || userID <= 0 {
		response.Fail(c, http.StatusUnauthorized, response.CodeTokenUserInvalid, "无效的用户信息")
		return
	}

	roleCodes, permissionCodes, err := h.rbacService.GetUserAuthorization(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err, response.CodeRBACFailed, "读取用户授权信息失败")
		return
	}

	response.Success(c, response.AuthorizationInfoResponse{
		RoleCodes:       roleCodes,
		PermissionCodes: permissionCodes,
	})
}

// AssignUserRolesHandler godoc
// @Summary 给用户分配角色
// @Tags admin-rbac
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param body body request.AssignUserRolesRequest true "角色编码列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/admin/users/{id}/roles [put]
func (h *RBACHandler) AssignUserRolesHandler(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "无效的用户ID")
		return
	}

	var req request.AssignUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}

	if err := h.rbacService.AssignRolesToUser(c.Request.Context(), userID, req.RoleCodes); err != nil {
		handleError(c, err, response.CodeRBACFailed, "分配用户角色失败")
		return
	}

	response.Success(c, nil)
}
