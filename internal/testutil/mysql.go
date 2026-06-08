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
