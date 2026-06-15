package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"go-user-system/config"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const mainSQLDriverName = "go_user_system_main_test"

var registerMainSQLDriverOnce sync.Once

type mainSQLDriver struct{}

func (mainSQLDriver) Open(name string) (driver.Conn, error) {
	return mainSQLConn{}, nil
}

type mainSQLConn struct{}

func (mainSQLConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (mainSQLConn) Close() error {
	return nil
}

func (mainSQLConn) Begin() (driver.Tx, error) {
	return mainSQLTx{}, nil
}

type mainSQLTx struct{}

func (mainSQLTx) Commit() error {
	return nil
}

func (mainSQLTx) Rollback() error {
	return nil
}

func openMainGormDB(t *testing.T) *gorm.DB {
	t.Helper()

	registerMainSQLDriverOnce.Do(func() {
		sql.Register(mainSQLDriverName, mainSQLDriver{})
	})

	sqlDB, err := sql.Open(mainSQLDriverName, "main")
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

type fakeAppServer struct {
	listenErr      error
	shutdownErr    error
	stop           chan struct{}
	closeStop      sync.Once
	shutdownCalled bool
}

func (s *fakeAppServer) ListenAndServe() error {
	if s.stop != nil {
		<-s.stop
	}
	return s.listenErr
}

func (s *fakeAppServer) Shutdown(ctx context.Context) error {
	s.shutdownCalled = true
	if s.stop != nil {
		s.closeStop.Do(func() {
			close(s.stop)
		})
	}
	return s.shutdownErr
}

func baseRunDeps(t *testing.T) appDeps {
	t.Helper()

	return appDeps{
		loadEnv: func() {},
		loadConfig: func(path string) (*config.Config, error) {
			return &config.Config{
				Server: config.ServerConfig{Port: 8080},
			}, nil
		},
		initJWTKey: func(cfg *config.Config) error {
			return nil
		},
		initDB: func(cfg *config.Config) (*gorm.DB, error) {
			return openMainGormDB(t), nil
		},
		setupRouter: func(db *gorm.DB, logger *slog.Logger) http.Handler {
			return http.NewServeMux()
		},
		newServer: func(addr string, handler http.Handler, cfg config.HttpServerConfig) appServer {
			return &fakeAppServer{listenErr: http.ErrServerClosed}
		},
		notify:          func(c chan<- os.Signal, sig ...os.Signal) {},
		shutdownTimeout: time.Second,
	}
}

func TestDefaultAppDepsProvidesDependencies(t *testing.T) {
	deps := defaultAppDeps()

	if deps.loadEnv == nil || deps.loadConfig == nil || deps.initJWTKey == nil || deps.initDB == nil {
		t.Fatal("expected default dependencies to be initialized")
	}
	if deps.shutdownTimeout != 10*time.Second {
		t.Fatalf("expected shutdown timeout 10s, got %s", deps.shutdownTimeout)
	}
	if deps.setupRouter(nil, nil) == nil {
		t.Fatal("expected default router")
	}
	if deps.newServer(":0", http.NewServeMux(), config.HttpServerConfig{}) == nil {
		t.Fatal("expected default http server")
	}
}

func TestMainRunsWithInjectedDefaultDependencies(t *testing.T) {
	oldGetDefaultAppDeps := getDefaultAppDeps
	oldFatalf := fatalf
	t.Cleanup(func() {
		getDefaultAppDeps = oldGetDefaultAppDeps
		fatalf = oldFatalf
	})

	getDefaultAppDeps = func() appDeps {
		return baseRunDeps(t)
	}
	fatalf = func(format string, v ...interface{}) {
		t.Fatalf("fatalf should not be called: "+format, v...)
	}

	main()
}

func TestMainCallsFatalWhenRunFails(t *testing.T) {
	oldGetDefaultAppDeps := getDefaultAppDeps
	oldFatalf := fatalf
	t.Cleanup(func() {
		getDefaultAppDeps = oldGetDefaultAppDeps
		fatalf = oldFatalf
	})

	expectedErr := errors.New("load failed")
	getDefaultAppDeps = func() appDeps {
		deps := baseRunDeps(t)
		deps.loadConfig = func(path string) (*config.Config, error) {
			return nil, expectedErr
		}
		return deps
	}

	fatalCalled := false
	fatalf = func(format string, v ...interface{}) {
		fatalCalled = true
		panic("fatal called")
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected fatal panic")
		}
		if !fatalCalled {
			t.Fatal("expected fatalf to be called")
		}
	}()

	main()
}

func TestRunReturnsLoadConfigError(t *testing.T) {
	expectedErr := errors.New("load failed")
	deps := baseRunDeps(t)
	deps.loadConfig = func(path string) (*config.Config, error) {
		return nil, expectedErr
	}

	err := run(deps)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected load config error, got %v", err)
	}
}

func TestRunReturnsInitJWTError(t *testing.T) {
	expectedErr := errors.New("jwt failed")
	deps := baseRunDeps(t)
	deps.initJWTKey = func(cfg *config.Config) error {
		return expectedErr
	}

	err := run(deps)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected init jwt error, got %v", err)
	}
}

func TestRunReturnsInitDBError(t *testing.T) {
	expectedErr := errors.New("db failed")
	deps := baseRunDeps(t)
	deps.initDB = func(cfg *config.Config) (*gorm.DB, error) {
		return nil, expectedErr
	}

	err := run(deps)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected init db error, got %v", err)
	}
}

func TestRunReturnsDatabaseHandleError(t *testing.T) {
	deps := baseRunDeps(t)
	deps.initDB = func(cfg *config.Config) (*gorm.DB, error) {
		return &gorm.DB{Config: &gorm.Config{}}, nil
	}

	err := run(deps)

	if err == nil || !strings.Contains(err.Error(), "get database handle failed") {
		t.Fatalf("expected database handle error, got %v", err)
	}
}

func TestRunReturnsListenError(t *testing.T) {
	expectedErr := errors.New("listen failed")
	deps := baseRunDeps(t)
	deps.newServer = func(addr string, handler http.Handler, cfg config.HttpServerConfig) appServer {
		return &fakeAppServer{listenErr: expectedErr}
	}

	err := run(deps)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected listen error, got %v", err)
	}
}

func TestRunReturnsNilWhenServerStopsNormally(t *testing.T) {
	deps := baseRunDeps(t)

	err := run(deps)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestRunReturnsShutdownError(t *testing.T) {
	expectedErr := errors.New("shutdown failed")
	server := &fakeAppServer{
		listenErr:   http.ErrServerClosed,
		shutdownErr: expectedErr,
		stop:        make(chan struct{}),
	}
	deps := baseRunDeps(t)
	deps.newServer = func(addr string, handler http.Handler, cfg config.HttpServerConfig) appServer {
		return server
	}
	deps.notify = func(c chan<- os.Signal, sig ...os.Signal) {
		c <- syscall.SIGTERM
	}

	err := run(deps)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected shutdown error, got %v", err)
	}
	if !server.shutdownCalled {
		t.Fatal("expected server shutdown to be called")
	}
}

func TestRunShutsDownGracefullyOnSignal(t *testing.T) {
	server := &fakeAppServer{
		listenErr: http.ErrServerClosed,
		stop:      make(chan struct{}),
	}
	deps := baseRunDeps(t)
	deps.shutdownTimeout = 0
	deps.newServer = func(addr string, handler http.Handler, cfg config.HttpServerConfig) appServer {
		return server
	}
	deps.notify = func(c chan<- os.Signal, sig ...os.Signal) {
		c <- syscall.SIGTERM
	}

	err := run(deps)

	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !server.shutdownCalled {
		t.Fatal("expected server shutdown to be called")
	}
}
