package service

import (
	"context"
	"errors"
	"testing"

	"go-user-system/internal/model"
	"go-user-system/internal/request"
)

func TestBootstrapAdminValidatesCredentials(t *testing.T) {
	userService := NewUserService(nil)

	if err := userService.BootstrapAdmin(context.Background(), request.RegisterRequest{
		Username: "ab",
		Password: "password1234",
	}); !errors.Is(err, ErrUsernameTooShort) {
		t.Fatalf("expected ErrUsernameTooShort, got %v", err)
	}
	if err := userService.BootstrapAdmin(context.Background(), request.RegisterRequest{
		Username: "admin",
		Password: "short",
	}); !errors.Is(err, ErrPasswordTooShortOrTooLong) {
		t.Fatalf("expected ErrPasswordTooShortOrTooLong, got %v", err)
	}
}

func TestBootstrapAdminRequiresDatabase(t *testing.T) {
	err := NewUserService(nil).BootstrapAdmin(context.Background(), request.RegisterRequest{
		Username: "admin",
		Password: "password1234",
	})
	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestBootstrapAdminIntegrationCreatesOnlyAdministrator(t *testing.T) {
	db := prepareUserServiceIntegrationDB(t)
	userService := NewUserService(db)
	ctx := context.Background()

	if err := userService.BootstrapAdmin(ctx, request.RegisterRequest{
		Username: "bootstrap_admin",
		Password: "password1234",
	}); err != nil {
		t.Fatalf("bootstrap administrator failed: %v", err)
	}

	var roleCodes []string
	if err := db.Table("user_roles AS ur").
		Joins("JOIN roles AS r ON r.id = ur.role_id").
		Joins("JOIN users AS u ON u.id = ur.user_id").
		Where("u.username = ?", "bootstrap_admin").
		Order("r.code ASC").
		Pluck("r.code", &roleCodes).Error; err != nil {
		t.Fatalf("list administrator roles failed: %v", err)
	}
	if len(roleCodes) != 2 || roleCodes[0] != model.RoleCodeAdmin || roleCodes[1] != model.RoleCodeUser {
		t.Fatalf("expected roles [admin user], got %v", roleCodes)
	}

	err := userService.BootstrapAdmin(ctx, request.RegisterRequest{
		Username: "second_admin",
		Password: "password1234",
	})
	if !errors.Is(err, ErrAdminAlreadyBootstrapped) {
		t.Fatalf("expected ErrAdminAlreadyBootstrapped, got %v", err)
	}
}
