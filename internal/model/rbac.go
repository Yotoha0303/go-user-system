package model

import "time"

const (
	RoleCodeAdmin = "admin"
	RoleCodeUser  = "user"

	PermissionProfileRead       = "profile:read"
	PermissionProfileUpdate     = "profile:update"
	PermissionPasswordUpdate    = "password:update"
	PermissionAdminRolesRead    = "admin:roles:read"
	PermissionAdminPermsRead    = "admin:permissions:read"
	PermissionAdminUserRoleEdit = "admin:user_roles:update"
)

type Role struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"size:64;not null;uniqueIndex:uk_roles_code" json:"code"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Role) TableName() string {
	return "roles"
}

type Permission struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"size:128;not null;uniqueIndex:uk_permissions_code" json:"code"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Method    string    `gorm:"size:16;not null;uniqueIndex:uk_permissions_method_path" json:"method"`
	Path      string    `gorm:"size:255;not null;uniqueIndex:uk_permissions_method_path" json:"path"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Permission) TableName() string {
	return "permissions"
}

type UserRole struct {
	UserID    int64     `gorm:"primaryKey;not null" json:"user_id"`
	RoleID    int64     `gorm:"primaryKey;not null" json:"role_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

type RolePermission struct {
	RoleID       int64     `gorm:"primaryKey;not null" json:"role_id"`
	PermissionID int64     `gorm:"primaryKey;not null" json:"permission_id"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}
