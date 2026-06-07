package migration

import (
	"errors"
	"go-user-system/internal/testutil"
	"os"
	"path/filepath"
	"testing"

	"gorm.io/gorm"
)

func TestRunMigrationsRequiresDatabase(t *testing.T) {
	err := RunMigrations(nil, "migrations")

	if !errors.Is(err, ErrMigrationDatabaseMissing) {
		t.Fatalf("expected ErrMigrationDatabaseMissing, got %v", err)
	}
}

func TestCollectMigrationFilesReturnsSortedUpMigrations(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"002_add_status.up.sql":     "SELECT 2;",
		"001_create_users.up.sql":   "SELECT 1;",
		"001_create_users.down.sql": "DROP TABLE users;",
		"README.md":                 "ignored",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write file %s failed: %v", name, err)
		}
	}

	migrations, err := collectMigrationFiles(dir)
	if err != nil {
		t.Fatalf("collect migration files failed: %v", err)
	}

	if len(migrations) != 2 {
		t.Fatalf("expected 2 up migrations, got %d", len(migrations))
	}
	if migrations[0].version != "001_create_users" {
		t.Fatalf("expected first version 001_create_users, got %s", migrations[0].version)
	}
	if migrations[1].version != "002_add_status" {
		t.Fatalf("expected second version 002_add_status, got %s", migrations[1].version)
	}
}

func TestSplitSQLStatementsSkipsEmptyStatements(t *testing.T) {
	statements := splitSQLStatements(`
CREATE TABLE users (id BIGINT);

CREATE INDEX idx_users_id ON users(id);
`)

	if len(statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(statements))
	}
	if statements[0] != "CREATE TABLE users (id BIGINT)" {
		t.Fatalf("unexpected first statement: %s", statements[0])
	}
	if statements[1] != "CREATE INDEX idx_users_id ON users(id)" {
		t.Fatalf("unexpected second statement: %s", statements[1])
	}
}

func TestFindDirUpwardFindsParentMigrationDir(t *testing.T) {
	root := t.TempDir()
	migrationDir := filepath.Join(root, "migrations")
	child := filepath.Join(root, "cmd")

	if err := os.Mkdir(migrationDir, 0o700); err != nil {
		t.Fatalf("mkdir migrations failed: %v", err)
	}
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatalf("mkdir child failed: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore wd failed: %v", err)
		}
	}()

	if err := os.Chdir(child); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	path, ok := findDirUpward("migrations")
	if !ok {
		t.Fatal("expected migrations directory to be found")
	}
	if path != migrationDir {
		t.Fatalf("expected path %s, got %s", migrationDir, path)
	}
}

func prepareMigrationIntegrationDB(t *testing.T, tableNames ...string) *gorm.DB {
	t.Helper()

	db := testutil.OpenMySQL(t)
	testutil.ResetTables(t, db, append([]string{"schema_migrations"}, tableNames...)...)

	t.Cleanup(func() {
		testutil.ResetTables(t, db, append([]string{"schema_migrations"}, tableNames...)...)
		testutil.CloseMySQL(t, db)
	})

	return db
}

func writeMigrationFile(t *testing.T, dir string, name string, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write migration file %s failed: %v", name, err)
	}
}

func TestRunMigrationsIntegrationCreatesSchemaTableAndSkipsAppliedVersions(t *testing.T) {
	db := prepareMigrationIntegrationDB(t, "migration_items")
	dir := t.TempDir()

	writeMigrationFile(t, dir, "001_create_migration_items.up.sql", `
CREATE TABLE migration_items (
  id BIGINT NOT NULL PRIMARY KEY,
  name VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
INSERT INTO migration_items (id, name) VALUES (1, 'first');
`)

	if err := RunMigrations(db, dir); err != nil {
		t.Fatalf("run migrations failed: %v", err)
	}
	if !db.Migrator().HasTable(schemaMigrationsTable) {
		t.Fatal("expected schema_migrations table to be created")
	}

	var versionCount int64
	if err := db.Raw("SELECT COUNT(1) FROM schema_migrations WHERE version = ?", "001_create_migration_items").Scan(&versionCount).Error; err != nil {
		t.Fatalf("count migration version failed: %v", err)
	}
	if versionCount != 1 {
		t.Fatalf("expected migration version to be recorded once, got %d", versionCount)
	}

	if err := RunMigrations(db, dir); err != nil {
		t.Fatalf("run migrations second time failed: %v", err)
	}

	var itemCount int64
	if err := db.Table("migration_items").Count(&itemCount).Error; err != nil {
		t.Fatalf("count migration items failed: %v", err)
	}
	if itemCount != 1 {
		t.Fatalf("expected idempotent migration item count 1, got %d", itemCount)
	}
}

func TestRunMigrationsIntegrationDoesNotRecordFailedMigration(t *testing.T) {
	db := prepareMigrationIntegrationDB(t)
	dir := t.TempDir()

	writeMigrationFile(t, dir, "001_bad.up.sql", `
INSERT INTO missing_migration_table (id) VALUES (1);
`)

	err := RunMigrations(db, dir)
	if err == nil {
		t.Fatal("expected migration failure")
	}

	var versionCount int64
	if err := db.Raw("SELECT COUNT(1) FROM schema_migrations WHERE version = ?", "001_bad").Scan(&versionCount).Error; err != nil {
		t.Fatalf("count failed migration version failed: %v", err)
	}
	if versionCount != 0 {
		t.Fatalf("expected failed migration not to be recorded, got %d", versionCount)
	}
}
