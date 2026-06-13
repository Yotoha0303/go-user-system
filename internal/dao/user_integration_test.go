package dao

import (
	"context"
	"errors"
	"go-user-system/internal/model"
	"go-user-system/internal/testutil"
	"testing"
	"time"

	"gorm.io/gorm"
)

func prepareUserDAOIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testutil.OpenMySQL(t)
	testutil.ResetTables(t, db, "schema_migrations", "users")

	t.Cleanup(func() {
		testutil.ResetTables(t, db, "schema_migrations", "users")
		testutil.CloseMySQL(t, db)
	})

	return db
}

func TestUserDAOIntegrationCreateReadAndUpdateUser(t *testing.T) {
	db := prepareUserDAOIntegrationDB(t)
	ctx := context.Background()
	username := testutil.UniqueName(t, "dao_user")

	user := model.User{
		Username:     username,
		PasswordHash: "bcrypt_hash",
		Nickname:     "alice",
		Status:       model.UserStatusActive,
	}
	if err := CreateUser(ctx, db, &user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected created user id to be set")
	}

	byUsername, err := GetUserByUsername(ctx, db, username)
	if err != nil {
		t.Fatalf("get user by username failed: %v", err)
	}
	if byUsername.ID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, byUsername.ID)
	}

	byID, err := GetUserByID(ctx, db, user.ID)
	if err != nil {
		t.Fatalf("get user by id failed: %v", err)
	}
	if byID.Username != username {
		t.Fatalf("expected username %s, got %s", username, byID.Username)
	}

	if err := UpdateNicknameByID(ctx, db, user.ID, "new-name"); err != nil {
		t.Fatalf("update nickname failed: %v", err)
	}
	updatedUser, err := GetUserByID(ctx, db, user.ID)
	if err != nil {
		t.Fatalf("get updated user failed: %v", err)
	}
	if updatedUser.Nickname != "new-name" {
		t.Fatalf("expected nickname new-name, got %s", updatedUser.Nickname)
	}

	lastLoginAt := time.Now().Truncate(time.Millisecond)
	if err := UpdateLastLoginAtByID(ctx, db, user.ID, lastLoginAt); err != nil {
		t.Fatalf("update last login at failed: %v", err)
	}
	loggedInUser, err := GetUserByID(ctx, db, user.ID)
	if err != nil {
		t.Fatalf("get logged in user failed: %v", err)
	}
	if loggedInUser.LastLoginAt == nil || loggedInUser.LastLoginAt.IsZero() {
		t.Fatal("expected last_login_at to be stored")
	}
}

func TestUserDAOIntegrationReturnsNotFoundForMissingUser(t *testing.T) {
	db := prepareUserDAOIntegrationDB(t)
	ctx := context.Background()

	_, err := GetUserByUsername(ctx, db, testutil.UniqueName(t, "missing_user"))
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound by username, got %v", err)
	}

	_, err = GetUserByID(ctx, db, 999999999)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound by id, got %v", err)
	}
}

func TestUserDAOIntegrationHonorsCanceledContext(t *testing.T) {
	db := prepareUserDAOIntegrationDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := GetUserByUsername(ctx, db, testutil.UniqueName(t, "canceled_user"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
