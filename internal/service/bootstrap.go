package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go-user-system/internal/model"
	"go-user-system/internal/repository"
	"go-user-system/internal/request"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BootstrapAdmin creates the first administrator after migrations have seeded RBAC roles.
func (s *UserService) BootstrapAdmin(ctx context.Context, req request.RegisterRequest) error {
	username := strings.TrimSpace(req.Username)
	if len(username) < 3 {
		return ErrUsernameTooShort
	}
	if err := validatePassword(req.Password); err != nil {
		return err
	}
	if err := s.ensureDB(); err != nil {
		return err
	}
	if s.rbacRepo == nil {
		return ErrRBACNotInitialized
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash administrator password: %w", err)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var adminRole model.Role
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code = ?", model.RoleCodeAdmin).
			First(&adminRole).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRBACNotInitialized
		}
		if err != nil {
			return fmt.Errorf("lock administrator role: %w", err)
		}

		var adminCount int64
		if err := tx.Model(&model.UserRole{}).Where("role_id = ?", adminRole.ID).Count(&adminCount).Error; err != nil {
			return fmt.Errorf("count administrators: %w", err)
		}
		if adminCount > 0 {
			return ErrAdminAlreadyBootstrapped
		}

		if _, err := s.store.GetUserByUsername(ctx, tx, username); err == nil {
			return ErrUsernameAlreadyExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check administrator username: %w", err)
		}

		user := model.User{
			Username:     username,
			PasswordHash: string(hashBytes),
			Nickname:     username,
			Status:       model.UserStatusActive,
		}
		if err := s.store.CreateUser(ctx, tx, &user); err != nil {
			return fmt.Errorf("create administrator: %w", err)
		}
		if err := s.rbacRepo.AssignRoleToUserByCode(ctx, tx, user.ID, model.RoleCodeUser); err != nil {
			if errors.Is(err, repository.ErrRoleNotFound) {
				return ErrRBACNotInitialized
			}
			return fmt.Errorf("assign user role: %w", err)
		}
		if err := s.rbacRepo.AssignRoleToUserByCode(ctx, tx, user.ID, model.RoleCodeAdmin); err != nil {
			return fmt.Errorf("assign administrator role: %w", err)
		}
		return nil
	})
}
