package database

import (
	"errors"
	"go-user-system/config"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func validDBConfig() *config.Config {
	return &config.Config{
		MySQL: config.MySQLConfig{
			Host:     "127.0.0.1",
			Port:     "3306",
			User:     "root",
			Database: "go_user_system",
		},
	}
}

func TestBuildDSNRequiresCompleteConfig(t *testing.T) {
	_, err := buildDSN(&config.Config{}, "secret")

	if err == nil {
		t.Fatal("expected missing database config error")
	}
}

func TestBuildDSNRequiresPassword(t *testing.T) {
	_, err := buildDSN(validDBConfig(), "")

	if err == nil {
		t.Fatal("expected missing password error")
	}
}

func TestBuildDSNBuildsMySQLConnectionString(t *testing.T) {
	dsn, err := buildDSN(validDBConfig(), "secret")

	if err != nil {
		t.Fatalf("build dsn failed: %v", err)
	}
	expectedParts := []string{
		"root:secret@tcp(127.0.0.1:3306)/go_user_system",
		"charset=utf8mb4",
		"parseTime=True",
		"loc=Local",
	}
	for _, part := range expectedParts {
		if !strings.Contains(dsn, part) {
			t.Fatalf("expected dsn to contain %s, got %s", part, dsn)
		}
	}
}

func TestInitDBReturnsConfigErrorBeforeOpeningConnection(t *testing.T) {
	t.Setenv("DB_PASSWORD", "")

	_, err := InitDB(validDBConfig())

	if err == nil {
		t.Fatal("expected database config error")
	}
}

func TestInitDBUsesOpenMySQL(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")

	oldOpenMySQL := openMySQL
	t.Cleanup(func() {
		openMySQL = oldOpenMySQL
	})

	var gotDSN string
	expectedDB := &gorm.DB{}
	openMySQL = func(dsn string) (*gorm.DB, error) {
		gotDSN = dsn
		return expectedDB, nil
	}

	db, err := InitDB(validDBConfig())

	if err != nil {
		t.Fatalf("init db failed: %v", err)
	}
	if db != expectedDB {
		t.Fatal("expected InitDB to return db from openMySQL")
	}
	if !strings.Contains(gotDSN, "root:secret@tcp(127.0.0.1:3306)/go_user_system") {
		t.Fatalf("unexpected dsn: %s", gotDSN)
	}
}

func TestInitDBReturnsOpenError(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")

	oldOpenMySQL := openMySQL
	t.Cleanup(func() {
		openMySQL = oldOpenMySQL
	})

	openErr := errors.New("open failed")
	openMySQL = func(dsn string) (*gorm.DB, error) {
		return nil, openErr
	}

	_, err := InitDB(validDBConfig())

	if !errors.Is(err, openErr) {
		t.Fatalf("expected open error, got %v", err)
	}
}

func TestDefaultOpenMySQLReturnsConnectionError(t *testing.T) {
	_, err := openMySQL("root:secret@tcp(127.0.0.1:1)/go_user_system?timeout=1ms")

	if err == nil {
		t.Fatal("expected connection error")
	}
}
