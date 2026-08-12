package database

import (
	"strings"
	"testing"
)

func TestMigrationConfigFromEnv(t *testing.T) {
	values := map[string]string{
		"DB_HOST":     "mysql",
		"DB_PORT":     "3306",
		"DB_USER":     "go_user_system",
		"DB_PASSWORD": "secret",
		"DB_NAME":     "go_user_system",
	}

	cfg, err := MigrationConfigFromEnv(func(key string) string { return values[key] })
	if err != nil {
		t.Fatalf("load migration config failed: %v", err)
	}
	if cfg.Host != "mysql" || cfg.Port != "3306" || cfg.User != "go_user_system" || cfg.Password != "secret" || cfg.Database != "go_user_system" {
		t.Fatalf("unexpected migration config: %+v", cfg)
	}
}

func TestMigrationConfigFromEnvRejectsMissingValues(t *testing.T) {
	_, err := MigrationConfigFromEnv(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected missing migration config error")
	}
	for _, name := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("expected error to contain %s, got %v", name, err)
		}
	}
}
