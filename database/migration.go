package database

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrMigrationDatabaseMissing = errors.New("migration database is nil")
	ErrMigrationDirNotFound     = errors.New("migration directory not found")
)

type migrationFile struct {
	version string
	path    string
}

const schemaMigrationsTable = "schema_migrations"

func RunMigrations(db *gorm.DB, dir string) error {
	if db == nil {
		return ErrMigrationDatabaseMissing
	}

	migrationDir, ok := findDirUpward(dir)
	if !ok {
		return fmt.Errorf("%w: %s", ErrMigrationDirNotFound, dir)
	}

	migrations, err := collectMigrationFiles(migrationDir)
	if err != nil {
		return err
	}

	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	for _, migration := range migrations {
		applied, err := isMigrationApplied(db, migration.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := applyMigration(db, migration); err != nil {
			return err
		}
	}

	return nil
}

func ensureSchemaMigrationsTable(db *gorm.DB) error {
	if db.Migrator().HasTable(schemaMigrationsTable) {
		return nil
	}

	return db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version VARCHAR(255) NOT NULL,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
`).Error
}

func isMigrationApplied(db *gorm.DB, version string) (bool, error) {
	var count int64
	err := db.Raw("SELECT COUNT(1) FROM schema_migrations WHERE version = ?", version).Scan(&count).Error
	return count > 0, err
}

func applyMigration(db *gorm.DB, migration migrationFile) error {
	content, err := os.ReadFile(migration.path)
	if err != nil {
		return fmt.Errorf("read migration %s failed: %w", migration.version, err)
	}

	statements := splitSQLStatements(string(content))
	if len(statements) == 0 {
		return fmt.Errorf("migration %s has no executable sql", migration.version)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		applied, err := isMigrationApplied(tx, migration.version)
		if err != nil {
			return fmt.Errorf("check migration %s failed: %w", migration.version, err)
		}
		if applied {
			return nil
		}

		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("apply migration %s failed: %w", migration.version, err)
			}
		}

		if err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.version).Error; err != nil {
			return fmt.Errorf("record migration %s failed: %w", migration.version, err)
		}

		return nil
	})
}

func collectMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migration dir failed: %w", err)
	}

	migrations := make([]migrationFile, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}

		version := strings.TrimSuffix(name, ".up.sql")
		migrations = append(migrations, migrationFile{
			version: version,
			path:    filepath.Join(dir, name),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

func splitSQLStatements(sql string) []string {
	parts := strings.Split(sql, ";")
	statements := make([]string, 0, len(parts))

	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement == "" {
			continue
		}
		statements = append(statements, statement)
	}

	return statements
}

func findDirUpward(name string) (string, bool) {
	if filepath.IsAbs(name) {
		info, err := os.Stat(name)
		return name, err == nil && info.IsDir()
	}

	for _, dir := range migrationSearchStartDirs() {
		if path, ok := findDirUpwardFrom(dir, name); ok {
			return path, true
		}
	}

	return "", false
}

func migrationSearchStartDirs() []string {
	dirs := make([]string, 0, 3)
	seen := make(map[string]struct{})

	addDir := func(dir string) {
		if dir == "" {
			return
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return
		}
		if _, ok := seen[absDir]; ok {
			return
		}
		seen[absDir] = struct{}{}
		dirs = append(dirs, absDir)
	}

	if dir, err := os.Getwd(); err == nil {
		addDir(dir)
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		addDir(filepath.Dir(file))
	}

	if exePath, err := os.Executable(); err == nil {
		addDir(filepath.Dir(exePath))
	}

	return dirs
}

func findDirUpwardFrom(startDir string, name string) (string, bool) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
