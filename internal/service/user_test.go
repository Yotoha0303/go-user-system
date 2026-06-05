package service

import (
	"errors"
	"go-user-system/internal/request"
	"strings"
	"testing"
)

func TestRegisterValidatesUsernameLength(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.Register(request.RegisterRequest{
		Username: "ab",
		Password: "123456",
	})

	if !errors.Is(err, ErrUsernameTooShort) {
		t.Fatalf("expected ErrUsernameTooShort, got %v", err)
	}
}

func TestRegisterValidatesPasswordLength(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.Register(request.RegisterRequest{
		Username: "alice",
		Password: "12345",
	})

	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestRegisterRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.Register(request.RegisterRequest{
		Username: "alice",
		Password: "123456",
	})

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestLoginRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	_, err := userService.Login(request.LoginRequest{
		Username: "alice",
		Password: "123456",
	})

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestGetProfileValidatesUserID(t *testing.T) {
	userService := NewUserService(nil)

	_, err := userService.GetProfile(0)

	if !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestGetProfileRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	_, err := userService.GetProfile(1)

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestUpdateNicknameValidatesUserID(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(0, "alice")

	if !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestUpdateNicknameValidatesEmptyNickname(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(1, "   ")

	if !errors.Is(err, ErrNicknameEmpty) {
		t.Fatalf("expected ErrNicknameEmpty, got %v", err)
	}
}

func TestUpdateNicknameValidatesNicknameLength(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(1, strings.Repeat("a", 65))

	if !errors.Is(err, ErrNicknameTooLong) {
		t.Fatalf("expected ErrNicknameTooLong, got %v", err)
	}
}

func TestUpdateNicknameRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(1, "alice")

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}
