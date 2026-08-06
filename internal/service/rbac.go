package service

import (
	"context"
	"errors"
	"go-user-system/internal/apperror"
	"go-user-system/internal/model"
	"go-user-system/internal/repository"
	"go-user-system/internal/response"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

type RBACService struct {
	db   *gorm.DB
	repo repository.RBACRepository
}

func NewRBACService(db *gorm.DB) *RBACService {
	return &RBACService{
		db:   db,
		repo: repository.NewGormRBACRepository(),
	}
}

func (s *RBACService) ensureDB() error {
	if s == nil || s.db == nil {
		return ErrDatabaseNotInitialized
	}
	return nil
}

func (s *RBACService) HasPermission(ctx context.Context, userID int64, permissionCode string) (bool, error) {
	if userID <= 0 || strings.TrimSpace(permissionCode) == "" {
		return false, ErrPermissionDenied
	}
	if err := s.ensureDB(); err != nil {
		return false, err
	}

	allowed, err := s.repo.UserHasPermission(ctx, s.db, userID, permissionCode)
	if err != nil {
		return false, apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRBACFailed,
			"权限校验失败",
			err,
		)
	}
	return allowed, nil
}

func (s *RBACService) ListRoles(ctx context.Context) ([]model.Role, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	roles, err := s.repo.ListRoles(ctx, s.db)
	if err != nil {
		return nil, apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRBACFailed,
			"读取角色列表失败",
			err,
		)
	}
	return roles, nil
}

func (s *RBACService) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	permissions, err := s.repo.ListPermissions(ctx, s.db)
	if err != nil {
		return nil, apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRBACFailed,
			"读取权限列表失败",
			err,
		)
	}
	return permissions, nil
}

func (s *RBACService) GetUserAuthorization(ctx context.Context, userID int64) ([]string, []string, error) {
	if userID <= 0 {
		return nil, nil, ErrInvalidUserID
	}
	if err := s.ensureDB(); err != nil {
		return nil, nil, err
	}

	roleCodes, err := s.repo.ListUserRoleCodes(ctx, s.db, userID)
	if err != nil {
		return nil, nil, apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRBACFailed,
			"读取用户角色失败",
			err,
		)
	}

	permissionCodes, err := s.repo.ListUserPermissionCodes(ctx, s.db, userID)
	if err != nil {
		return nil, nil, apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRBACFailed,
			"读取用户权限失败",
			err,
		)
	}

	return roleCodes, permissionCodes, nil
}

func (s *RBACService) AssignRolesToUser(ctx context.Context, userID int64, roleCodes []string) error {
	if userID <= 0 {
		return ErrInvalidUserID
	}
	if len(roleCodes) == 0 {
		return ErrRoleNotFound
	}
	if err := s.ensureDB(); err != nil {
		return err
	}

	normalizedRoleCodes := make([]string, 0, len(roleCodes))
	seen := make(map[string]struct{}, len(roleCodes))
	for _, roleCode := range roleCodes {
		roleCode = strings.TrimSpace(roleCode)
		if roleCode == "" {
			return ErrRoleNotFound
		}
		if _, ok := seen[roleCode]; ok {
			continue
		}
		seen[roleCode] = struct{}{}
		normalizedRoleCodes = append(normalizedRoleCodes, roleCode)
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.ReplaceUserRolesByCodes(ctx, tx, userID, normalizedRoleCodes)
	})
	if errors.Is(err, repository.ErrRoleNotFound) {
		return ErrRoleNotFound
	}
	if err != nil {
		return apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRBACFailed,
			"分配用户角色失败",
			err,
		)
	}
	return nil
}
