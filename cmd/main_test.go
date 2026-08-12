package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"go-user-system/config"
	"go-user-system/internal/auth"
	"go-user-system/internal/authstate"
	"go-user-system/internal/request"
	"go-user-system/router"
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

var (
	registerMainSQLDriverOnce sync.Once
)

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
		loadEnv: func() error {
			return nil
		},
		loadConfig: func(path string) (*config.Config, error) {
			return &config.Config{
				Server: config.ServerConfig{Port: 8080},
				JWT: config.JWTConfig{
					ExpireHours:              24,
					AccessTokenExpireMinutes: 15,
					RefreshTokenExpireHours:  168,
				},
			}, nil
		},
		initDB: func(cfg *config.Config) (*gorm.DB, error) {
			return openMainGormDB(t), nil
		},
		openMigrationDB: func(ctx context.Context) (*sql.DB, error) {
			db := openMainGormDB(t)
			return db.DB()
		},
		newAuthStateStore: func(ctx context.Context, cfg config.RedisConfig) (authstate.Store, error) {
			return authstate.NewMemoryStore(), nil
		},
		newTokenManager: func(secret string, issuer string, accessTTL time.Duration, refreshTTL time.Duration) (*auth.TokenManager, error) {
			return &auth.TokenManager{}, nil
		},
		setupRouter: func(db *gorm.DB, logger *slog.Logger, tokenManager *auth.TokenManager, runtime router.AuthRuntime) http.Handler {
			return http.NewServeMux()
		},
		bootstrapAdmin: func(ctx context.Context, db *gorm.DB, req request.RegisterRequest) error {
			return nil
		},
		migrateUp: func(ctx context.Context, db *sql.DB, dir string) error {
			return nil
		},
		getenv: func(key string) string { return "" },
		newServer: func(addr string, handler http.Handler, cfg config.HttpServerConfig) appServer {
			return &fakeAppServer{listenErr: http.ErrServerClosed}
		},
		notify:          func(c chan<- os.Signal, sig ...os.Signal) {},
		shutdownTimeout: time.Second,
	}
}

func TestDefaultAppDepsProvidesDependencies(t *testing.T) {
	deps := defaultAppDeps()

	if deps.loadEnv == nil || deps.loadConfig == nil || deps.initDB == nil || deps.openMigrationDB == nil {
		t.Fatal("expected default dependencies to be initialized")
	}
	if deps.shutdownTimeout != 10*time.Second {
		t.Fatalf("expected shutdown timeout 10s, got %s", deps.shutdownTimeout)
	}
	if deps.newAuthStateStore == nil {
		t.Fatal("expected authentication state store factory")
	}
	if deps.setupRouter(nil, nil, &auth.TokenManager{}, router.AuthRuntime{}) == nil {
		t.Fatal("expected default router")
	}
	if deps.bootstrapAdmin == nil || deps.migrateUp == nil || deps.getenv == nil {
		t.Fatal("expected administrator bootstrap dependencies")
	}
	if deps.newServer(":0", http.NewServeMux(), config.HttpServerConfig{}) == nil {
		t.Fatal("expected default http server")
	}
}

func TestRunMigrateUpUsesMigrationsDirectory(t *testing.T) {
	deps := baseRunDeps(t)
	called := false
	deps.migrateUp = func(ctx context.Context, db *sql.DB, dir string) error {
		called = true
		if db == nil || dir != "migrations" {
			t.Fatalf("unexpected migration arguments: db=%v dir=%q", db, dir)
		}
		return nil
	}

	if err := runMigrateUp(deps); err != nil {
		t.Fatalf("run migration failed: %v", err)
	}
	if !called {
		t.Fatal("expected migration runner to be called")
	}
}

func TestRunMigrateUpReturnsMigrationError(t *testing.T) {
	expectedErr := errors.New("migration failed")
	deps := baseRunDeps(t)
	deps.migrateUp = func(ctx context.Context, db *sql.DB, dir string) error {
		return expectedErr
	}

	err := runMigrateUp(deps)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected migration error, got %v", err)
	}
}

func TestRunBootstrapAdminRequiresCredentials(t *testing.T) {
	deps := baseRunDeps(t)

	err := runBootstrapAdmin(deps)

	if err == nil || !strings.Contains(err.Error(), "BOOTSTRAP_ADMIN_USERNAME") {
		t.Fatalf("expected missing bootstrap credentials error, got %v", err)
	}
}

func TestRunBootstrapAdminCreatesAdministrator(t *testing.T) {
	deps := baseRunDeps(t)
	deps.getenv = func(key string) string {
		values := map[string]string{
			"BOOTSTRAP_ADMIN_USERNAME": "admin-user",
			"BOOTSTRAP_ADMIN_PASSWORD": "strong-password",
		}
		return values[key]
	}
	called := false
	deps.bootstrapAdmin = func(ctx context.Context, db *gorm.DB, req request.RegisterRequest) error {
		called = true
		if req.Username != "admin-user" || req.Password != "strong-password" {
			t.Fatalf("unexpected bootstrap request: %+v", req)
		}
		return nil
	}

	if err := runBootstrapAdmin(deps); err != nil {
		t.Fatalf("bootstrap administrator failed: %v", err)
	}
	if !called {
		t.Fatal("expected bootstrap service to be called")
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

func TestRunReturnsAuthenticationStateStoreError(t *testing.T) {
	expectedErr := errors.New("redis unavailable")
	deps := baseRunDeps(t)
	deps.newAuthStateStore = func(ctx context.Context, cfg config.RedisConfig) (authstate.Store, error) {
		return nil, expectedErr
	}

	err := run(deps)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected authentication state store error, got %v", err)
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

func TestRunPassesRequestTimeoutToRouter(t *testing.T) {
	expectedTimeout := 7 * time.Second
	deps := baseRunDeps(t)
	deps.loadConfig = func(path string) (*config.Config, error) {
		return &config.Config{
			Server: config.ServerConfig{Port: 8080},
			JWT: config.JWTConfig{
				ExpireHours:              24,
				AccessTokenExpireMinutes: 15,
				RefreshTokenExpireHours:  168,
			},
			HttpServer: config.HttpServer{
				Server: config.HttpServerConfig{Timeout: expectedTimeout},
			},
		}, nil
	}
	var actualTimeout time.Duration
	deps.setupRouter = func(db *gorm.DB, logger *slog.Logger, tokenManager *auth.TokenManager, runtime router.AuthRuntime) http.Handler {
		actualTimeout = runtime.RequestTimeout
		return http.NewServeMux()
	}

	if err := run(deps); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if actualTimeout != expectedTimeout {
		t.Fatalf("expected router timeout %s, got %s", expectedTimeout, actualTimeout)
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
