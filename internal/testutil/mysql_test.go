package testutil

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const testutilSQLDriverName = "go_user_system_testutil_test"

var registerTestutilSQLDriverOnce sync.Once

type testutilSQLDriver struct{}

func (testutilSQLDriver) Open(name string) (driver.Conn, error) {
	return testutilSQLConn{}, nil
}

type testutilSQLConn struct{}

func (testutilSQLConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (testutilSQLConn) Close() error {
	return nil
}

func (testutilSQLConn) Begin() (driver.Tx, error) {
	return testutilSQLTx{}, nil
}

func (testutilSQLConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func (testutilSQLConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	upperQuery := strings.ToUpper(query)
	if strings.Contains(upperQuery, "DATABASE()") {
		return &testutilSQLRows{
			columns: []string{"database"},
			values:  []driver.Value{"go_user_test"},
		}, nil
	}
	if strings.Contains(upperQuery, "GET_LOCK") {
		return &testutilSQLRows{
			columns: []string{"lock_result"},
			values:  []driver.Value{int64(1)},
		}, nil
	}
	return &testutilSQLRows{
		columns: []string{"value"},
		values:  []driver.Value{int64(0)},
	}, nil
}

type testutilSQLTx struct{}

func (testutilSQLTx) Commit() error {
	return nil
}

func (testutilSQLTx) Rollback() error {
	return nil
}

type testutilSQLRows struct {
	columns []string
	values  []driver.Value
	read    bool
}

func (r *testutilSQLRows) Columns() []string {
	return r.columns
}

func (r *testutilSQLRows) Close() error {
	return nil
}

func (r *testutilSQLRows) Next(dest []driver.Value) error {
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

func openTestutilGormDB(t *testing.T) *gorm.DB {
	t.Helper()

	registerTestutilSQLDriverOnce.Do(func() {
		sql.Register(testutilSQLDriverName, testutilSQLDriver{})
	})

	sqlDB, err := sql.Open(testutilSQLDriverName, "testutil")
	if err != nil {
		t.Fatalf("open sql db failed: %v", err)
	}

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open gorm db failed: %v", err)
	}
	return db
}

func TestOpenMySQLSkipsWhenDSNIsMissing(t *testing.T) {
	t.Setenv(TestDatabaseDSNEnv, "")

	OpenMySQL(t)
}

func TestOpenMySQLInitializesTestDatabase(t *testing.T) {
	oldOpenMySQLDB := openMySQLDB
	t.Cleanup(func() {
		openMySQLDB = oldOpenMySQLDB
	})

	var gotDSN string
	openMySQLDB = func(dsn string) (*gorm.DB, error) {
		gotDSN = dsn
		return openTestutilGormDB(t), nil
	}

	t.Setenv(TestDatabaseDSNEnv, "fake-test-dsn")

	db := OpenMySQL(t)
	if gotDSN != "fake-test-dsn" {
		t.Fatalf("expected dsn fake-test-dsn, got %s", gotDSN)
	}
	CloseMySQL(t, db)
}

func TestDefaultOpenMySQLDBReturnsConnectionError(t *testing.T) {
	_, err := openMySQLDB("root:secret@tcp(127.0.0.1:1)/go_user_test?timeout=1ms")

	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestCloseMySQLIgnoresNilDatabase(t *testing.T) {
	CloseMySQL(t, nil)
}

func TestCloseMySQLClosesDatabase(t *testing.T) {
	CloseMySQL(t, openTestutilGormDB(t))
}

func TestResetTablesIgnoresEmptyTableList(t *testing.T) {
	ResetTables(t, nil)
}

func TestResetTablesDropsProvidedTables(t *testing.T) {
	db := openTestutilGormDB(t)
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	ResetTables(t, db, "schema_migrations", "users")
}

func TestUniqueNameUsesPrefix(t *testing.T) {
	name := UniqueName(t, "user")

	if !strings.HasPrefix(name, "user_") {
		t.Fatalf("expected prefix user_, got %s", name)
	}
	if len(name) != len("user_")+8 {
		t.Fatalf("expected 8 hex characters, got %s", name)
	}
}

func TestQuoteIdentifierAcceptsSafeIdentifier(t *testing.T) {
	got := quoteIdentifier(t, "schema_migrations_2026")

	if got != "`schema_migrations_2026`" {
		t.Fatalf("expected quoted identifier, got %s", got)
	}
}

func TestAssertTestDatabaseAcceptsDatabaseWithTestInName(t *testing.T) {
	db := openTestutilGormDB(t)
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	assertTestDatabase(t, db)
}

func TestAcquireIntegrationLockSucceeds(t *testing.T) {
	db := openTestutilGormDB(t)
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	acquireIntegrationLock(t, db)
}
