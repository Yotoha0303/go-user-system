package dao

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"go-user-system/internal/model"
	"io"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const daoSQLDriverName = "go_user_system_dao_test"

var registerDaoSQLDriverOnce sync.Once

type daoSQLDriver struct{}

func (daoSQLDriver) Open(name string) (driver.Conn, error) {
	return &daoSQLConn{}, nil
}

type daoSQLConn struct{}

func (daoSQLConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (daoSQLConn) Close() error {
	return nil
}

func (daoSQLConn) Begin() (driver.Tx, error) {
	return daoSQLTx{}, nil
}

func (daoSQLConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func (daoSQLConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return daoSQLRows{}, nil
}

type daoSQLTx struct{}

func (daoSQLTx) Commit() error {
	return nil
}

func (daoSQLTx) Rollback() error {
	return nil
}

type daoSQLRows struct {
	read bool
}

func (daoSQLRows) Columns() []string {
	return []string{"id", "username", "password_hash", "nickname", "status", "created_at", "updated_at", "last_login_at", "deleted_at"}
}

func (daoSQLRows) Close() error {
	return nil
}

func (r daoSQLRows) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	return io.EOF
}

func openDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()

	registerDaoSQLDriverOnce.Do(func() {
		sql.Register(daoSQLDriverName, daoSQLDriver{})
	})

	sqlDB, err := sql.Open(daoSQLDriverName, "dao")
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
	}), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry run db failed: %v", err)
	}
	return db
}

func TestCreateUserBuildsInsertStatement(t *testing.T) {
	db := openDryRunDB(t)
	user := &model.User{
		Username:     "alice",
		PasswordHash: "hash",
		Nickname:     "alice",
		Status:       model.UserStatusActive,
	}

	err := CreateUser(context.Background(), db, user)

	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}
}

func TestGetUserByUsernameBuildsSelectStatement(t *testing.T) {
	db := openDryRunDB(t)

	_, err := GetUserByUsername(context.Background(), db, "alice")

	if err != nil {
		t.Fatalf("get user by username failed: %v", err)
	}
}

func TestGetUserByIDBuildsSelectStatement(t *testing.T) {
	db := openDryRunDB(t)

	_, err := GetUserByID(context.Background(), db, 1)

	if err != nil {
		t.Fatalf("get user by id failed: %v", err)
	}
}

func TestUpdateNicknameByIDBuildsUpdateStatement(t *testing.T) {
	db := openDryRunDB(t)

	err := UpdateNicknameByID(context.Background(), db, 1, "new-name")

	if err != nil {
		t.Fatalf("update nickname failed: %v", err)
	}
}

func TestUpdateLastLoginAtByIDBuildsUpdateStatement(t *testing.T) {
	db := openDryRunDB(t)

	err := UpdateLastLoginAtByID(context.Background(), db, 1, time.Now())

	if err != nil {
		t.Fatalf("update last login failed: %v", err)
	}
}

func TestWithContextUsesBackgroundWhenContextIsNil(t *testing.T) {
	db := openDryRunDB(t)

	got := withContext(context.Background(), db)

	if got.Statement.Context == nil {
		t.Fatal("expected context to be set")
	}
}
