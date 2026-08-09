package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const TestDatabaseDSNEnv = "TEST_DATABASE_DSN"
const mysqlIntegrationLockName = "go_user_system_integration_tests"

const createUsersTableSQL = `CREATE TABLE users (
	id BIGINT NOT NULL AUTO_INCREMENT,
	username VARCHAR(64) NOT NULL,
	password_hash VARCHAR(255) NOT NULL,
	nickname VARCHAR(64) NOT NULL DEFAULT '',
	status TINYINT NOT NULL DEFAULT 1,
	auth_version BIGINT NOT NULL DEFAULT 1,
	created_at DATETIME(3) NULL DEFAULT NULL,
	updated_at DATETIME(3) NULL DEFAULT NULL,
	last_login_at DATETIME(3) NULL DEFAULT NULL,
	deleted_at DATETIME(3) NULL DEFAULT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY idx_username (username),
	KEY idx_users_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`

//nolint:gosec // Schema column names are not credentials.
const createRefreshTokensTableSQL = `CREATE TABLE refresh_tokens (
	id BIGINT NOT NULL AUTO_INCREMENT,
	user_id BIGINT NOT NULL,
	jti VARCHAR(64) NOT NULL,
	family_id VARCHAR(64) NOT NULL,
	token_hash CHAR(64) NOT NULL,
	expires_at DATETIME(3) NOT NULL,
	revoked_at DATETIME(3) NULL DEFAULT NULL,
	revoked_reason VARCHAR(32) NULL DEFAULT NULL,
	replaced_by_jti VARCHAR(64) NULL DEFAULT NULL,
	created_at DATETIME(3) NULL DEFAULT NULL,
	updated_at DATETIME(3) NULL DEFAULT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uk_refresh_tokens_jti (jti),
	UNIQUE KEY uk_refresh_tokens_hash (token_hash),
	KEY idx_refresh_tokens_user_id (user_id),
	KEY idx_refresh_tokens_family_id (family_id),
	KEY idx_refresh_tokens_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`

const createRolesTableSQL = `CREATE TABLE roles (
	id BIGINT NOT NULL AUTO_INCREMENT,
	code VARCHAR(64) NOT NULL,
	name VARCHAR(64) NOT NULL,
	created_at DATETIME(3) NULL DEFAULT NULL,
	updated_at DATETIME(3) NULL DEFAULT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uk_roles_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`

const createUserRolesTableSQL = `CREATE TABLE user_roles (
	user_id BIGINT NOT NULL,
	role_id BIGINT NOT NULL,
	created_at DATETIME(3) NULL DEFAULT NULL,
	PRIMARY KEY (user_id, role_id),
	KEY idx_user_roles_role_id (role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`

const seedRolesSQL = `INSERT INTO roles (code, name, created_at, updated_at)
VALUES
	('admin', 'admin', NOW(3), NOW(3)),
	('user', 'user', NOW(3), NOW(3))`

var openMySQLDB = func(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func OpenMySQL(t testing.TB) *gorm.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv(TestDatabaseDSNEnv))
	if dsn == "" {
		t.Skipf("set %s to run MySQL integration tests", TestDatabaseDSNEnv)
	}

	db, err := openMySQLDB(dsn)
	if err != nil {
		t.Fatalf("open test mysql failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	assertTestDatabase(t, db)
	acquireIntegrationLock(t, db)

	return db
}

func CloseMySQL(t testing.TB, db *gorm.DB) {
	t.Helper()

	if db == nil {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db failed: %v", err)
	}
}

func ResetTables(t testing.TB, db *gorm.DB, tableNames ...string) {
	t.Helper()

	if len(tableNames) == 0 {
		return
	}

	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		t.Fatalf("disable foreign key checks failed: %v", err)
	}
	defer func() {
		if err := db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
			t.Fatalf("enable foreign key checks failed: %v", err)
		}
	}()

	for _, tableName := range tableNames {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteIdentifier(t, tableName))).Error; err != nil {
			t.Fatalf("drop table %s failed: %v", tableName, err)
		}
	}
}

func CreateUsersTable(t testing.TB, db *gorm.DB) {
	t.Helper()

	if err := db.Exec(createUsersTableSQL).Error; err != nil {
		t.Fatalf("create users table failed: %v", err)
	}
}

func CreateRefreshTokensTable(t testing.TB, db *gorm.DB) {
	t.Helper()

	if err := db.Exec(createRefreshTokensTableSQL).Error; err != nil {
		t.Fatalf("create refresh tokens table failed: %v", err)
	}
}

func CreateRoleAssignmentTables(t testing.TB, db *gorm.DB) {
	t.Helper()

	statements := []struct {
		name string
		sql  string
	}{
		{name: "roles", sql: createRolesTableSQL},
		{name: "user_roles", sql: createUserRolesTableSQL},
		{name: "role seeds", sql: seedRolesSQL},
	}
	for _, statement := range statements {
		if err := db.Exec(statement.sql).Error; err != nil {
			t.Fatalf("create %s failed: %v", statement.name, err)
		}
	}
}

func UniqueName(t testing.TB, prefix string) string {
	t.Helper()

	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		t.Fatalf("generate random bytes failed: %v", err)
	}

	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(randomBytes))
}

func assertTestDatabase(t testing.TB, db *gorm.DB) {
	t.Helper()

	var databaseName string
	if err := db.Raw("SELECT DATABASE()").Scan(&databaseName).Error; err != nil {
		t.Fatalf("read current database failed: %v", err)
	}

	if !strings.Contains(strings.ToLower(databaseName), "test") {
		t.Fatalf("refusing to run integration tests on non-test database %q", databaseName)
	}
}

func acquireIntegrationLock(t testing.TB, db *gorm.DB) {
	t.Helper()

	var lockResult int
	if err := db.Raw("SELECT GET_LOCK(?, 30)", mysqlIntegrationLockName).Scan(&lockResult).Error; err != nil {
		t.Fatalf("acquire mysql integration lock failed: %v", err)
	}
	if lockResult != 1 {
		t.Fatalf("acquire mysql integration lock timed out")
	}
}

func quoteIdentifier(t testing.TB, identifier string) string {
	t.Helper()

	if identifier == "" {
		t.Fatal("empty sql identifier")
	}

	for _, character := range identifier {
		if character == '_' || unicode.IsLetter(character) || unicode.IsDigit(character) {
			continue
		}
		t.Fatalf("invalid sql identifier %q", identifier)
	}

	return "`" + identifier + "`"
}
