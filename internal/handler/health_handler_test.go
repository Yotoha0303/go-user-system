package handler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"go-user-system/internal/response"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const healthSQLDriverName = "go_user_system_health_test"

var registerHealthSQLDriverOnce sync.Once
var healthSQLPingCount atomic.Int32

type healthSQLDriver struct{}

func (healthSQLDriver) Open(name string) (driver.Conn, error) {
	conn := &healthSQLConn{}
	if name == "ping-fail" {
		conn.pingErr = errors.New("ping failed")
	}
	return conn, nil
}

type healthSQLConn struct {
	pingErr error
}

func (c *healthSQLConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (c *healthSQLConn) Close() error {
	return nil
}

func (c *healthSQLConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction is not supported")
}

func (c *healthSQLConn) Ping(ctx context.Context) error {
	healthSQLPingCount.Add(1)
	return c.pingErr
}

func openHealthGormDB(t *testing.T, name string) *gorm.DB {
	t.Helper()

	registerHealthSQLDriverOnce.Do(func() {
		sql.Register(healthSQLDriverName, healthSQLDriver{})
	})

	sqlDB, err := sql.Open(healthSQLDriverName, name)
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
		t.Fatalf("open gorm db failed: %v", err)
	}
	return db
}

func TestPingHandlerReturnsSuccessMessage(t *testing.T) {
	healthHandler := NewHealthHandler(nil)

	recorder := performJSONRequest(healthHandler.PingHandler, http.MethodGet, "/ping", "")
	body := decodeResponse(t, recorder)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
	}
}

func TestLivezHandlerReturnsAliveStatus(t *testing.T) {
	healthHandler := NewHealthHandler(nil)

	recorder := performJSONRequest(healthHandler.LivezHandler, http.MethodGet, "/livez", "")
	body := decodeResponse(t, recorder)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
	}
}

func TestReadyzHandlerFailsWhenDatabaseIsNil(t *testing.T) {
	healthHandler := NewHealthHandler(nil)

	recorder := performJSONRequest(healthHandler.ReadyzHandler, http.MethodGet, "/readyz", "")
	body := decodeResponse(t, recorder)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
	if body.Code != response.CodeReadinessFailed {
		t.Fatalf("expected code %d, got %d", response.CodeReadinessFailed, body.Code)
	}
}

func TestReadyzHandlerFailsWhenDatabaseHandleIsInvalid(t *testing.T) {
	healthHandler := NewHealthHandler(&gorm.DB{})

	recorder := performJSONRequest(healthHandler.ReadyzHandler, http.MethodGet, "/readyz", "")
	body := decodeResponse(t, recorder)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
	if body.Code != response.CodeReadinessFailed {
		t.Fatalf("expected code %d, got %d", response.CodeReadinessFailed, body.Code)
	}
}

func TestReadyzHandlerFailsWhenDatabaseHandleReturnsError(t *testing.T) {
	healthHandler := NewHealthHandler(&gorm.DB{Config: &gorm.Config{}})

	recorder := performJSONRequest(healthHandler.ReadyzHandler, http.MethodGet, "/readyz", "")
	body := decodeResponse(t, recorder)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
	if body.Code != response.CodeReadinessFailed {
		t.Fatalf("expected code %d, got %d", response.CodeReadinessFailed, body.Code)
	}
}

func TestReadyzHandlerFailsWhenDatabasePingFails(t *testing.T) {
	healthHandler := NewHealthHandler(openHealthGormDB(t, "ping-fail"))

	recorder := performJSONRequest(healthHandler.ReadyzHandler, http.MethodGet, "/readyz", "")
	body := decodeResponse(t, recorder)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
	if body.Code != response.CodeReadinessFailed {
		t.Fatalf("expected code %d, got %d", response.CodeReadinessFailed, body.Code)
	}
}

func TestReadyzHandlerReturnsReadyWhenDatabasePings(t *testing.T) {
	healthSQLPingCount.Store(0)
	healthHandler := NewHealthHandler(openHealthGormDB(t, "ready"))

	recorder := performJSONRequest(healthHandler.ReadyzHandler, http.MethodGet, "/readyz", "")
	body := decodeResponse(t, recorder)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
	}
	if got := healthSQLPingCount.Load(); got != 1 {
		t.Fatalf("expected database ping once, got %d", got)
	}
}
