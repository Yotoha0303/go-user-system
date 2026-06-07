package service

import (
	"context"
	"errors"
	"go-user-system/internal/dao"
	"go-user-system/internal/model"
	"go-user-system/internal/request"
	"go-user-system/internal/testutil"
	"go-user-system/pkg/migration"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestRegisterValidatesUsernameLength(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "ab",
		Password: "123456",
	})

	if !errors.Is(err, ErrUsernameTooShort) {
		t.Fatalf("expected ErrUsernameTooShort, got %v", err)
	}
}

func TestRegisterValidatesPasswordLength(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "alice",
		Password: "12345",
	})

	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestRegisterRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "alice",
		Password: "123456",
	})

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestLoginRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	_, err := userService.Login(context.Background(), request.LoginRequest{
		Username: "alice",
		Password: "123456",
	})

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestGetProfileValidatesUserID(t *testing.T) {
	userService := NewUserService(nil)

	_, err := userService.GetProfile(context.Background(), 0)

	if !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestGetProfileRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	_, err := userService.GetProfile(context.Background(), 1)

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestUpdateNicknameValidatesUserID(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(context.Background(), 0, "alice")

	if !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestUpdateNicknameValidatesEmptyNickname(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(context.Background(), 1, "   ")

	if !errors.Is(err, ErrNicknameEmpty) {
		t.Fatalf("expected ErrNicknameEmpty, got %v", err)
	}
}

func TestUpdateNicknameValidatesNicknameLength(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(context.Background(), 1, strings.Repeat("a", 65))

	if !errors.Is(err, ErrNicknameTooLong) {
		t.Fatalf("expected ErrNicknameTooLong, got %v", err)
	}
}

func TestUpdateNicknameRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(context.Background(), 1, "alice")

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func prepareUserServiceIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testutil.OpenMySQL(t)
	testutil.ResetTables(t, db, "schema_migrations", "users")
	if err := migration.RunMigrations(db, "migrations"); err != nil {
		t.Fatalf("run migrations failed: %v", err)
	}

	t.Cleanup(func() {
		testutil.ResetTables(t, db, "schema_migrations", "users")
		testutil.CloseMySQL(t, db)
	})

	return db
}

func TestUserServiceIntegrationRegisterLoginProfileAndNickname(t *testing.T) {
	db := prepareUserServiceIntegrationDB(t)
	userService := NewUserService(db)
	ctx := context.Background()
	username := testutil.UniqueName(t, "svc_user")
	password := "password123"

	err := userService.Register(ctx, request.RegisterRequest{
		Username: "  " + username + "  ",
		Password: password,
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	storedUser, err := dao.GetUserByUsername(ctx, db, username)
	if err != nil {
		t.Fatalf("get registered user failed: %v", err)
	}
	if storedUser.PasswordHash == password {
		t.Fatal("expected password to be stored as hash, got plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedUser.PasswordHash), []byte(password)); err != nil {
		t.Fatalf("stored password hash does not match password: %v", err)
	}
	if storedUser.Nickname != username {
		t.Fatalf("expected nickname %s, got %s", username, storedUser.Nickname)
	}
	if storedUser.Status != model.UserStatusActive {
		t.Fatalf("expected active status, got %d", storedUser.Status)
	}

	err = userService.Register(ctx, request.RegisterRequest{
		Username: username,
		Password: password,
	})
	if !errors.Is(err, ErrUsernameAlreadyExists) {
		t.Fatalf("expected ErrUsernameAlreadyExists, got %v", err)
	}

	_, err = userService.Login(ctx, request.LoginRequest{
		Username: username,
		Password: "wrong-password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	loggedInUser, err := userService.Login(ctx, request.LoginRequest{
		Username: "  " + username + "  ",
		Password: password,
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loggedInUser.LastLoginAt == nil || loggedInUser.LastLoginAt.IsZero() {
		t.Fatal("expected login response to include last_login_at")
	}

	afterLoginUser, err := dao.GetUserByID(ctx, db, loggedInUser.ID)
	if err != nil {
		t.Fatalf("get user after login failed: %v", err)
	}
	if afterLoginUser.LastLoginAt == nil || afterLoginUser.LastLoginAt.IsZero() {
		t.Fatal("expected last_login_at to be persisted")
	}

	profileUser, err := userService.GetProfile(ctx, loggedInUser.ID)
	if err != nil {
		t.Fatalf("get profile failed: %v", err)
	}
	if profileUser.Username != username {
		t.Fatalf("expected profile username %s, got %s", username, profileUser.Username)
	}

	if err := userService.UpdateNickname(ctx, loggedInUser.ID, "  new-nickname  "); err != nil {
		t.Fatalf("update nickname failed: %v", err)
	}
	updatedUser, err := dao.GetUserByID(ctx, db, loggedInUser.ID)
	if err != nil {
		t.Fatalf("get updated user failed: %v", err)
	}
	if updatedUser.Nickname != "new-nickname" {
		t.Fatalf("expected nickname new-nickname, got %s", updatedUser.Nickname)
	}
}

func TestUserServiceIntegrationRejectsDisabledUser(t *testing.T) {
	db := prepareUserServiceIntegrationDB(t)
	userService := NewUserService(db)
	ctx := context.Background()
	username := testutil.UniqueName(t, "disabled_user")
	password := "password123"

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate password hash failed: %v", err)
	}

	disabledUser := model.User{
		Username:     username,
		PasswordHash: string(hashBytes),
		Nickname:     username,
		Status:       model.UserStatusDisabled,
	}
	if err := dao.CreateUser(ctx, db, &disabledUser); err != nil {
		t.Fatalf("create disabled user failed: %v", err)
	}

	_, err = userService.Login(ctx, request.LoginRequest{
		Username: username,
		Password: password,
	})
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("expected ErrUserDisabled on login, got %v", err)
	}

	_, err = userService.GetProfile(ctx, disabledUser.ID)
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("expected ErrUserDisabled on profile, got %v", err)
	}

	err = userService.UpdateNickname(ctx, disabledUser.ID, "new-nickname")
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("expected ErrUserDisabled on nickname update, got %v", err)
	}
}
