package repository

import (
	"context"
	"errors"
	"go-user-system/internal/model"
	"time"

	"gorm.io/gorm"
)

var ErrRoleNotFound = errors.New("role not found")

type RBACRepository interface {
	AssignRoleToUserByCode(ctx context.Context, db *gorm.DB, userID int64, roleCode string) error
	UserHasPermission(ctx context.Context, db *gorm.DB, userID int64, permissionCode string) (bool, error)
	ListRoles(ctx context.Context, db *gorm.DB) ([]model.Role, error)
	ListPermissions(ctx context.Context, db *gorm.DB) ([]model.Permission, error)
	ListUserRoleCodes(ctx context.Context, db *gorm.DB, userID int64) ([]string, error)
	ListUserPermissionCodes(ctx context.Context, db *gorm.DB, userID int64) ([]string, error)
	ReplaceUserRolesByCodes(ctx context.Context, db *gorm.DB, userID int64, roleCodes []string) error
}

type GormRBACRepository struct{}

func NewGormRBACRepository() GormRBACRepository {
	return GormRBACRepository{}
}

func (GormRBACRepository) AssignRoleToUserByCode(ctx context.Context, db *gorm.DB, userID int64, roleCode string) error {
	role, err := findRoleByCode(ctx, db, roleCode)
	if err != nil {
		return err
	}

	userRole := model.UserRole{
		UserID:    userID,
		RoleID:    role.ID,
		CreatedAt: time.Now(),
	}
	return withContext(ctx, db).
		Where("user_id = ? AND role_id = ?", userID, role.ID).
		FirstOrCreate(&userRole).
		Error
}

func (GormRBACRepository) UserHasPermission(ctx context.Context, db *gorm.DB, userID int64, permissionCode string) (bool, error) {
	var count int64
	err := withContext(ctx, db).
		Table("user_roles AS ur").
		Joins("JOIN role_permissions AS rp ON rp.role_id = ur.role_id").
		Joins("JOIN permissions AS p ON p.id = rp.permission_id").
		Where("ur.user_id = ? AND p.code = ?", userID, permissionCode).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (GormRBACRepository) ListRoles(ctx context.Context, db *gorm.DB) ([]model.Role, error) {
	var roles []model.Role
	err := withContext(ctx, db).Order("id ASC").Find(&roles).Error
	return roles, err
}

func (GormRBACRepository) ListPermissions(ctx context.Context, db *gorm.DB) ([]model.Permission, error) {
	var permissions []model.Permission
	err := withContext(ctx, db).Order("id ASC").Find(&permissions).Error
	return permissions, err
}

func (GormRBACRepository) ListUserRoleCodes(ctx context.Context, db *gorm.DB, userID int64) ([]string, error) {
	roleCodes := make([]string, 0)
	err := withContext(ctx, db).
		Table("user_roles AS ur").
		Joins("JOIN roles AS r ON r.id = ur.role_id").
		Where("ur.user_id = ?", userID).
		Distinct("r.code").
		Order("r.code ASC").
		Pluck("r.code", &roleCodes).
		Error
	return roleCodes, err
}

func (GormRBACRepository) ListUserPermissionCodes(ctx context.Context, db *gorm.DB, userID int64) ([]string, error) {
	permissionCodes := make([]string, 0)
	err := withContext(ctx, db).
		Table("user_roles AS ur").
		Joins("JOIN role_permissions AS rp ON rp.role_id = ur.role_id").
		Joins("JOIN permissions AS p ON p.id = rp.permission_id").
		Where("ur.user_id = ?", userID).
		Distinct("p.code").
		Order("p.code ASC").
		Pluck("p.code", &permissionCodes).
		Error
	return permissionCodes, err
}

func (GormRBACRepository) ReplaceUserRolesByCodes(ctx context.Context, db *gorm.DB, userID int64, roleCodes []string) error {
	roles := make([]model.Role, 0, len(roleCodes))
	for _, roleCode := range roleCodes {
		role, err := findRoleByCode(ctx, db, roleCode)
		if err != nil {
			return err
		}
		roles = append(roles, *role)
	}

	if err := withContext(ctx, db).Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
		return err
	}

	for _, role := range roles {
		userRole := model.UserRole{
			UserID:    userID,
			RoleID:    role.ID,
			CreatedAt: time.Now(),
		}
		if err := withContext(ctx, db).Create(&userRole).Error; err != nil {
			return err
		}
	}
	return nil
}

func findRoleByCode(ctx context.Context, db *gorm.DB, roleCode string) (*model.Role, error) {
	var role model.Role
	err := withContext(ctx, db).Where("code = ?", roleCode).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRoleNotFound
	}
	return &role, err
}
