package migration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"go-user-system/internal/testutil"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const migrationSQLDriverName = "go_user_system_migration_test"

var registerMigrationSQLDriverOnce sync.Once

type migrationSQLDriver struct{}

func (migrationSQLDriver) Open(name string) (driver.Conn, error) {
	return migrationSQLConn{mode: name}, nil
}

type migrationSQLConn struct {
	mode string
}

func (migrationSQLConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (migrationSQLConn) Close() error {
	return nil
}

func (migrationSQLConn) Begin() (driver.Tx, error) {
	return migrationSQLTx{}, nil
}

func (migrationSQLConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return migrationSQLTx{}, nil
}

func (c migrationSQLConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	upperQuery := strings.ToUpper(query)
	if c.mode == "exec-fail" {
		return nil, errors.New("exec failed")
	}
	if c.mode == "apply-exec-fail" && !strings.Contains(upperQuery, "CREATE TABLE IF NOT EXISTS SCHEMA_MIGRATIONS") {
		return nil, errors.New("apply exec failed")
	}
	if c.mode == "record-fail" && strings.Contains(upperQuery, "INSERT INTO SCHEMA_MIGRATIONS") {
		return nil, errors.New("record failed")
	}
	return driver.RowsAffected(1), nil
}

func (c migrationSQLConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	upperQuery := strings.ToUpper(query)
	if c.mode == "query-fail" {
		return nil, errors.New("query failed")
	}
	if c.mode == "migration-check-fail" && strings.Contains(upperQuery, "SCHEMA_MIGRATIONS WHERE VERSION") {
		return nil, errors.New("migration check failed")
	}
	if strings.Contains(strings.ToUpper(query), "DATABASE()") {
		return &migrationSQLRows{
			columns: []string{"database"},
			values:  []driver.Value{"dryrun"},
		}, nil
	}

	count := int64(0)
	if c.mode == "applied" {
		count = 1
	}
	return &migrationSQLRows{
		columns: []string{"count"},
		values:  []driver.Value{count},
	}, nil
}

type migrationSQLTx struct{}

func (migrationSQLTx) Commit() error {
	return nil
}

func (migrationSQLTx) Rollback() error {
	return nil
}

type migrationSQLRows struct {
	columns []string
	values  []driver.Value
	read    bool
}

func (r *migrationSQLRows) Columns() []string {
	return r.columns
}

func (r *migrationSQLRows) Close() error {
	return nil
}

func (r *migrationSQLRows) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	for i := range dest {
		if i < len(r.values) {
			dest[i] = r.values[i]
			continue
		}
		dest[i] = nil
	}
	r.read = true
	return nil
}

func openMigrationDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	return openMigrationFakeDB(t, "migration")
}

func openMigrationFakeDB(t *testing.T, name string) *gorm.DB {
	t.Helper()

	registerMigrationSQLDriverOnce.Do(func() {
		sql.Register(migrationSQLDriverName, migrationSQLDriver{})
	})

	sqlDB, err := sql.Open(migrationSQLDriverName, name)
	if err != nil {
		t.Fatalf("open sql db failed: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close sql db failed: %v", err)
		}
	})

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open dry run db failed: %v", err)
	}
	return db
}

func TestRunMigrationsRequiresDatabase(t *testing.T) {
	err := RunMigrations(nil, "migrations")

	if !errors.Is(err, ErrMigrationDatabaseMissing) {
		t.Fatalf("expected ErrMigrationDatabaseMissing, got %v", err)
	}
}

func TestRunMigrationsReturnsDirNotFound(t *testing.T) {
	err := RunMigrations(openMigrationDryRunDB(t), filepath.Join(t.TempDir(), "missing"))

	if !errors.Is(err, ErrMigrationDirNotFound) {
		t.Fatalf("expected ErrMigrationDirNotFound, got %v", err)
	}
}

func TestRunMigrationsWithEmptyDirectory(t *testing.T) {
	err := RunMigrations(openMigrationDryRunDB(t), t.TempDir())

	if err != nil {
		t.Fatalf("run migrations failed: %v", err)
	}
}

func TestRunMigrationsAppliesPendingMigrationWithFakeDB(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_create_items.up.sql", "CREATE TABLE items (id BIGINT);")

	err := RunMigrations(openMigrationDryRunDB(t), dir)

	if err != nil {
		t.Fatalf("run migrations failed: %v", err)
	}
}

func TestRunMigrationsSkipsAppliedMigrationWithFakeDB(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_create_items.up.sql", "CREATE TABLE items (id BIGINT);")

	err := RunMigrations(openMigrationFakeDB(t, "applied"), dir)

	if err != nil {
		t.Fatalf("run migrations failed: %v", err)
	}
}

func TestRunMigrationsReturnsEnsureSchemaTableError(t *testing.T) {
	err := RunMigrations(openMigrationFakeDB(t, "exec-fail"), t.TempDir())

	if err == nil || !strings.Contains(err.Error(), "exec failed") {
		t.Fatalf("expected ensure schema table error, got %v", err)
	}
}

func TestRunMigrationsReturnsMigrationCheckError(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_create_items.up.sql", "CREATE TABLE items (id BIGINT);")

	err := RunMigrations(openMigrationFakeDB(t, "migration-check-fail"), dir)

	if err == nil || !strings.Contains(err.Error(), "migration check failed") {
		t.Fatalf("expected migration check error, got %v", err)
	}
}

func TestRunMigrationsReturnsApplyMigrationError(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_create_items.up.sql", "CREATE TABLE items (id BIGINT);")

	err := RunMigrations(openMigrationFakeDB(t, "apply-exec-fail"), dir)

	if err == nil || !strings.Contains(err.Error(), "apply migration 001_create_items failed") {
		t.Fatalf("expected apply migration error, got %v", err)
	}
}

func TestEnsureSchemaMigrationsTableWithDryRunDB(t *testing.T) {
	err := ensureSchemaMigrationsTable(openMigrationDryRunDB(t))

	if err != nil {
		t.Fatalf("ensure schema migrations table failed: %v", err)
	}
}

func TestEnsureSchemaMigrationsTableSkipsExistingTable(t *testing.T) {
	err := ensureSchemaMigrationsTable(openMigrationFakeDB(t, "applied"))

	if err != nil {
		t.Fatalf("ensure schema migrations table failed: %v", err)
	}
}

func TestIsMigrationAppliedWithDryRunDB(t *testing.T) {
	applied, err := isMigrationApplied(openMigrationDryRunDB(t), "001_init")

	if err != nil {
		t.Fatalf("check migration applied failed: %v", err)
	}
	if applied {
		t.Fatal("expected dry run migration to be unapplied")
	}
}

func TestIsMigrationAppliedReturnsTrueWhenVersionExists(t *testing.T) {
	applied, err := isMigrationApplied(openMigrationFakeDB(t, "applied"), "001_init")

	if err != nil {
		t.Fatalf("check migration applied failed: %v", err)
	}
	if !applied {
		t.Fatal("expected migration to be applied")
	}
}

func TestApplyMigrationReturnsReadFileError(t *testing.T) {
	err := applyMigration(openMigrationDryRunDB(t), migrationFile{
		version: "001_missing",
		path:    filepath.Join(t.TempDir(), "001_missing.up.sql"),
	})

	if err == nil || !strings.Contains(err.Error(), "read migration 001_missing failed") {
		t.Fatalf("expected read migration error, got %v", err)
	}
}

func TestApplyMigrationRejectsEmptyMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001_empty.up.sql")
	if err := os.WriteFile(path, []byte("  ;\n ;"), 0o600); err != nil {
		t.Fatalf("write empty migration failed: %v", err)
	}

	err := applyMigration(openMigrationDryRunDB(t), migrationFile{
		version: "001_empty",
		path:    path,
	})

	if err == nil || !strings.Contains(err.Error(), "has no executable sql") {
		t.Fatalf("expected empty migration error, got %v", err)
	}
}

func TestApplyMigrationExecutesStatementsWithDryRunDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001_create_items.up.sql")
	if err := os.WriteFile(path, []byte(`
CREATE TABLE items (id BIGINT NOT NULL PRIMARY KEY);
INSERT INTO items (id) VALUES (1);
`), 0o600); err != nil {
		t.Fatalf("write migration failed: %v", err)
	}

	err := applyMigration(openMigrationDryRunDB(t), migrationFile{
		version: "001_create_items",
		path:    path,
	})

	if err != nil {
		t.Fatalf("apply migration failed: %v", err)
	}
}

func TestApplyMigrationSkipsAlreadyAppliedVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001_create_items.up.sql")
	if err := os.WriteFile(path, []byte("CREATE TABLE items (id BIGINT);"), 0o600); err != nil {
		t.Fatalf("write migration failed: %v", err)
	}

	err := applyMigration(openMigrationFakeDB(t, "applied"), migrationFile{
		version: "001_create_items",
		path:    path,
	})

	if err != nil {
		t.Fatalf("apply migration failed: %v", err)
	}
}

func TestApplyMigrationWrapsCheckError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001_create_items.up.sql")
	if err := os.WriteFile(path, []byte("CREATE TABLE items (id BIGINT);"), 0o600); err != nil {
		t.Fatalf("write migration failed: %v", err)
	}

	err := applyMigration(openMigrationFakeDB(t, "migration-check-fail"), migrationFile{
		version: "001_create_items",
		path:    path,
	})

	if err == nil || !strings.Contains(err.Error(), "check migration 001_create_items failed") {
		t.Fatalf("expected check migration error, got %v", err)
	}
}

func TestApplyMigrationWrapsExecError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001_create_items.up.sql")
	if err := os.WriteFile(path, []byte("CREATE TABLE items (id BIGINT);"), 0o600); err != nil {
		t.Fatalf("write migration failed: %v", err)
	}

	err := applyMigration(openMigrationFakeDB(t, "exec-fail"), migrationFile{
		version: "001_create_items",
		path:    path,
	})

	if err == nil || !strings.Contains(err.Error(), "apply migration 001_create_items failed") {
		t.Fatalf("expected apply migration error, got %v", err)
	}
}

func TestApplyMigrationWrapsRecordError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001_create_items.up.sql")
	if err := os.WriteFile(path, []byte("CREATE TABLE items (id BIGINT);"), 0o600); err != nil {
		t.Fatalf("write migration failed: %v", err)
	}

	err := applyMigration(openMigrationFakeDB(t, "record-fail"), migrationFile{
		version: "001_create_items",
		path:    path,
	})

	if err == nil || !strings.Contains(err.Error(), "record migration 001_create_items failed") {
		t.Fatalf("expected record migration error, got %v", err)
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

func TestCollectMigrationFilesSkipsDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "003_dir.up.sql"), 0o700); err != nil {
		t.Fatalf("mkdir migration-like directory failed: %v", err)
	}
	writeMigrationFile(t, dir, "001_create_users.up.sql", "SELECT 1;")

	migrations, err := collectMigrationFiles(dir)

	if err != nil {
		t.Fatalf("collect migration files failed: %v", err)
	}
	if len(migrations) != 1 {
		t.Fatalf("expected 1 migration, got %d", len(migrations))
	}
	if migrations[0].version != "001_create_users" {
		t.Fatalf("expected version 001_create_users, got %s", migrations[0].version)
	}
}

func TestCollectMigrationFilesReturnsReadDirError(t *testing.T) {
	_, err := collectMigrationFiles(filepath.Join(t.TempDir(), "missing"))

	if err == nil || !strings.Contains(err.Error(), "read migration dir failed") {
		t.Fatalf("expected read dir error, got %v", err)
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

func TestFindDirUpwardAcceptsAbsoluteDirectory(t *testing.T) {
	dir := t.TempDir()

	path, ok := findDirUpward(dir)

	if !ok {
		t.Fatal("expected absolute directory to be found")
	}
	if path != dir {
		t.Fatalf("expected path %s, got %s", dir, path)
	}
}

func TestFindDirUpwardRejectsAbsoluteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration-file")
	if err := os.WriteFile(path, []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	_, ok := findDirUpward(path)

	if ok {
		t.Fatal("expected absolute file to be rejected")
	}
}

func TestFindDirUpwardReturnsFalseForMissingRelativeDirectory(t *testing.T) {
	_, ok := findDirUpward("definitely_missing_migrations_for_test")

	if ok {
		t.Fatal("expected missing relative directory not to be found")
	}
}

func TestFindDirUpwardFromReturnsFalseWhenDirectoryIsMissing(t *testing.T) {
	_, ok := findDirUpwardFrom(t.TempDir(), "missing-migrations")

	if ok {
		t.Fatal("expected missing directory not to be found")
	}
}

func TestMigrationSearchStartDirsDeduplicatesWorkingDirectory(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("expected caller file")
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

	if err := os.Chdir(filepath.Dir(file)); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	dirs := migrationSearchStartDirs()
	seen := make(map[string]struct{})
	for _, dir := range dirs {
		if _, ok := seen[dir]; ok {
			t.Fatalf("expected deduplicated dirs, got duplicate %s in %v", dir, dirs)
		}
		seen[dir] = struct{}{}
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
