package dao

import (
	"errors"
	"go-user-system/internal/model"
	"go-user-system/internal/testutil"
	"go-user-system/pkg/migration"
	"testing"
	"time"

	"gorm.io/gorm"
)

func prepareUserDAOIntegrationDB(t *testing.T) *gorm.DB {
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

func TestUserDAOIntegrationCreateReadAndUpdateUser(t *testing.T) {
	db := prepareUserDAOIntegrationDB(t)
	username := testutil.UniqueName(t, "dao_user")

	user := model.User{
		Username:     username,
		PasswordHash: "bcrypt_hash",
		Nickname:     "alice",
		Status:       model.UserStatusActive,
	}
	if err := CreateUser(db, &user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected created user id to be set")
	}

	byUsername, err := GetUserByUsername(db, username)
	if err != nil {
		t.Fatalf("get user by username failed: %v", err)
	}
	if byUsername.ID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, byUsername.ID)
	}

	byID, err := GetUserByID(db, user.ID)
	if err != nil {
		t.Fatalf("get user by id failed: %v", err)
	}
	if byID.Username != username {
		t.Fatalf("expected username %s, got %s", username, byID.Username)
	}

	if err := UpdateNicknameByID(db, user.ID, "new-name"); err != nil {
		t.Fatalf("update nickname failed: %v", err)
	}
	updatedUser, err := GetUserByID(db, user.ID)
	if err != nil {
		t.Fatalf("get updated user failed: %v", err)
	}
	if updatedUser.Nickname != "new-name" {
		t.Fatalf("expected nickname new-name, got %s", updatedUser.Nickname)
	}

	lastLoginAt := time.Now().Truncate(time.Millisecond)
	if err := UpdateLastLoginAtByID(db, user.ID, lastLoginAt); err != nil {
		t.Fatalf("update last login at failed: %v", err)
	}
	loggedInUser, err := GetUserByID(db, user.ID)
	if err != nil {
		t.Fatalf("get logged in user failed: %v", err)
	}
	if loggedInUser.LastLoginAt == nil || loggedInUser.LastLoginAt.IsZero() {
		t.Fatal("expected last_login_at to be stored")
	}
}

func TestUserDAOIntegrationReturnsNotFoundForMissingUser(t *testing.T) {
	db := prepareUserDAOIntegrationDB(t)

	_, err := GetUserByUsername(db, testutil.UniqueName(t, "missing_user"))
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound by username, got %v", err)
	}

	_, err = GetUserByID(db, 999999999)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound by id, got %v", err)
	}
}
