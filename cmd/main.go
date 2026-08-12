package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-user-system/config"
	"go-user-system/internal/auth"
	"go-user-system/internal/authstate"
	"go-user-system/internal/buildinfo"
	"go-user-system/internal/request"
	"go-user-system/internal/service"
	"go-user-system/pkg/database"
	"go-user-system/pkg/redisclient"
	"go-user-system/router"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

type appServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

type appDeps struct {
	loadEnv           func() error
	loadConfig        func(path string) (*config.Config, error)
	initDB            func(cfg *config.Config) (*gorm.DB, error)
	openMigrationDB   func(ctx context.Context) (*sql.DB, error)
	newAuthStateStore func(ctx context.Context, cfg config.RedisConfig) (authstate.Store, error)
	newTokenManager   func(secret string, issuer string, accessTTL time.Duration, refreshTTL time.Duration) (*auth.TokenManager, error)
	setupRouter       func(db *gorm.DB, logger *slog.Logger, tokenManager *auth.TokenManager, runtime router.AuthRuntime) http.Handler
	bootstrapAdmin    func(ctx context.Context, db *gorm.DB, req request.RegisterRequest) error
	migrateUp         func(ctx context.Context, db *sql.DB, dir string) error
	getenv            func(key string) string
	newServer         func(addr string, handler http.Handler, cfg config.HttpServerConfig) appServer
	notify            func(c chan<- os.Signal, sig ...os.Signal)
	shutdownTimeout   time.Duration
}

func defaultAppDeps() appDeps {
	return appDeps{
		loadEnv:         config.LoadEnv,
		loadConfig:      config.Load,
		initDB:          database.InitDB,
		openMigrationDB: database.OpenMigrationDBFromEnv,
		newAuthStateStore: func(ctx context.Context, cfg config.RedisConfig) (authstate.Store, error) {
			if !cfg.Enabled {
				return authstate.NewMemoryStore(), nil
			}
			client, err := redisclient.New(ctx, cfg)
			if err != nil {
				return nil, err
			}
			return authstate.NewRedisStore(client), nil
		},
		newTokenManager: func(secret string, issuer string, accessTTL time.Duration, refreshTTL time.Duration) (*auth.TokenManager, error) {
			return auth.NewTokenManagerWithTTL(secret, issuer, accessTTL, refreshTTL)
		},
		setupRouter: func(db *gorm.DB, logger *slog.Logger, tokenManager *auth.TokenManager, runtime router.AuthRuntime) http.Handler {
			return router.SetupRouter(db, logger, tokenManager, runtime)
		},
		bootstrapAdmin: func(ctx context.Context, db *gorm.DB, req request.RegisterRequest) error {
			return service.NewUserService(db).BootstrapAdmin(ctx, req)
		},
		migrateUp: func(ctx context.Context, db *sql.DB, dir string) error {
			if err := goose.SetDialect("mysql"); err != nil {
				return err
			}
			return goose.UpContext(ctx, db, dir)
		},
		getenv: os.Getenv,
		newServer: func(addr string, router http.Handler, cfg config.HttpServerConfig) appServer {
			return &http.Server{
				Addr:              addr,
				Handler:           router,
				ReadTimeout:       cfg.ReadTimeOut,
				WriteTimeout:      cfg.WriteTimeout,
				IdleTimeout:       cfg.IdleTimeout,
				ReadHeaderTimeout: cfg.ReadHeaderTimeout,
				MaxHeaderBytes:    cfg.MaxHeaderBytesKib << 10,
			}
		},
		notify:          signal.Notify,
		shutdownTimeout: 10 * time.Second,
	}
}

var (
	getDefaultAppDeps = defaultAppDeps
	fatalf            = log.Fatalf
)

// @title go-user-system API
// @version 1.0.0
// @description 用户系统后端接口文档，包含 JWT Access/Refresh 双 Token 认证、RBAC 权限控制和统一响应结构。
// @host localhost:8082
// @BasePath /
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 输入 Bearer access_token，例如：Bearer eyJhbGciOi...
func main() {
	deps := getDefaultAppDeps()
	var err error
	switch {
	case len(os.Args) > 1 && os.Args[1] == "bootstrap-admin":
		err = runBootstrapAdmin(deps)
	case len(os.Args) > 1 && os.Args[1] == "migrate":
		if len(os.Args) != 3 || os.Args[2] != "up" {
			err = errors.New("usage: go-user-system migrate up")
		} else {
			err = runMigrateUp(deps)
		}
	default:
		err = run(deps)
	}
	if err != nil {
		fatalf("application failed: %v", err)
	}
}

func runMigrateUp(deps appDeps) error {
	if err := deps.loadEnv(); err != nil {
		return err
	}

	sqlDB, err := deps.openMigrationDB(context.Background())
	if err != nil {
		return fmt.Errorf("open migration database failed: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close database failed: %v", err)
		}
	}()

	if err := deps.migrateUp(context.Background(), sqlDB, "migrations"); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	return nil
}

func runBootstrapAdmin(deps appDeps) error {
	if err := deps.loadEnv(); err != nil {
		return err
	}

	username := deps.getenv("BOOTSTRAP_ADMIN_USERNAME")
	password := deps.getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if username == "" || password == "" {
		return errors.New("BOOTSTRAP_ADMIN_USERNAME and BOOTSTRAP_ADMIN_PASSWORD are required")
	}

	cfg, err := deps.loadConfig("config.yml")
	if err != nil {
		return fmt.Errorf("load config failed: %w", err)
	}
	db, err := deps.initDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get database handle failed: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close database failed: %v", err)
		}
	}()

	if err := deps.bootstrapAdmin(context.Background(), db, request.RegisterRequest{
		Username: username,
		Password: password,
	}); err != nil {
		return fmt.Errorf("bootstrap administrator failed: %w", err)
	}
	log.Printf("administrator %q bootstrapped", username)
	return nil
}

func run(deps appDeps) error {

	if err := deps.loadEnv(); err != nil {
		return err
	}

	cfg, err := deps.loadConfig("config.yml")
	if err != nil {
		return fmt.Errorf("load config failed: %w", err)
	}

	db, err := deps.initDB(cfg)

	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get database handle failed: %w", err)
	}

	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close database failed: %v", err)
		}
	}()

	slog := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	logger := slog
	build := buildinfo.Current()

	tokenManager, err := deps.newTokenManager(
		cfg.JWT.Secret,
		"go-user-system",
		time.Duration(cfg.JWT.AccessTokenExpireMinutes)*time.Minute,
		time.Duration(cfg.JWT.RefreshTokenExpireHours)*time.Hour,
	)

	if err != nil {
		return fmt.Errorf("new token manager failed: %w", err)
	}

	authStateStore, err := deps.newAuthStateStore(context.Background(), cfg.Redis)
	if err != nil {
		return fmt.Errorf("initialize authentication state store failed: %w", err)
	}
	defer func() {
		if err := authStateStore.Close(); err != nil {
			logger.Error("close authentication state store failed", "error", err)
		}
	}()

	registrationEnabled := cfg.Auth.RegistrationEnabled()
	r := deps.setupRouter(db, slog, tokenManager, router.AuthRuntime{
		StateStore: authStateStore,
		LoginRateLimit: service.LoginRateLimit{
			AccountLimit: cfg.Auth.LoginRateLimit.AccountLimit,
			IPLimit:      cfg.Auth.LoginRateLimit.IPLimit,
			Window:       cfg.Auth.LoginRateLimit.Window,
		},
		RegistrationEnabled: &registrationEnabled,
		SecureCookies:       cfg.Auth.RefreshCookie.Secure,
		TrustedProxies:      cfg.HttpServer.TrustedProxies,
		RequestTimeout:      cfg.HttpServer.Server.Timeout,
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)

	server := deps.newServer(
		addr,
		r,
		cfg.HttpServer.Server,
	)

	serverErr := make(chan error, 1)
	go func() {
		logger.Info(
			"server starting",
			"addr", addr,
			"version", build.Version,
			"commit", build.Commit,
			"build_time", build.BuildTime,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	quit := make(chan os.Signal, 1)
	deps.notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server run failed: %w", err)
		}
		return nil
	}

	logger.Info("server shutting down")
	if deps.shutdownTimeout == 0 {
		deps.shutdownTimeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), deps.shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}
	logger.Info("server stopped")
	return nil
}
