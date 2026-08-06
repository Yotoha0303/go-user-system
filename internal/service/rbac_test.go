package service

import (
	"context"
	"go-user-system/internal/model"
	"testing"

	"gorm.io/gorm"
)

type authorizationRepository struct {
	roleCodes       []string
	permissionCodes []string
	roleErr         error
	permissionErr   error
}

func (r *authorizationRepository) AssignRoleToUserByCode(context.Context, *gorm.DB, int64, string) error {
	return nil
}
func (r *authorizationRepository) UserHasPermission(context.Context, *gorm.DB, int64, string) (bool, error) {
	return false, nil
}
func (r *authorizationRepository) ListRoles(context.Context, *gorm.DB) ([]model.Role, error) {
	return nil, nil
}
func (r *authorizationRepository) ListPermissions(context.Context, *gorm.DB) ([]model.Permission, error) {
	return nil, nil
}
func (r *authorizationRepository) ListUserRoleCodes(context.Context, *gorm.DB, int64) ([]string, error) {
	return r.roleCodes, r.roleErr
}
func (r *authorizationRepository) ListUserPermissionCodes(context.Context, *gorm.DB, int64) ([]string, error) {
	return r.permissionCodes, r.permissionErr
}
func (r *authorizationRepository) ReplaceUserRolesByCodes(context.Context, *gorm.DB, int64, []string) error {
	return nil
}

func TestGetUserAuthorizationReturnsCodes(t *testing.T) {
	repo := &authorizationRepository{
		roleCodes:       []string{"admin", "user"},
		permissionCodes: []string{"profile:read", "profile:update"},
	}
	service := &RBACService{db: &gorm.DB{}, repo: repo}

	roleCodes, permissionCodes, err := service.GetUserAuthorization(context.Background(), 1)
	if err != nil {
		t.Fatalf("get authorization failed: %v", err)
	}
	if len(roleCodes) != 2 || len(permissionCodes) != 2 {
		t.Fatalf("unexpected authorization codes: roles=%v permissions=%v", roleCodes, permissionCodes)
	}
}
