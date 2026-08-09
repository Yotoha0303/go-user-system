package main

import (
	"context"
	"errors"
	"fmt"
	"go-user-system/config"
	"go-user-system/internal/auth"
	"go-user-system/internal/authstate"
	"go-user-system/internal/middleware"
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
	newAuthStateStore func(ctx context.Context, cfg config.RedisConfig) (authstate.Store, error)
	newTokenManager   func(secret string, issuer string, accessTTL time.Duration, refreshTTL time.Duration) (*auth.TokenManager, error)
	setupRouter       func(db *gorm.DB, logger *slog.Logger, tokenManager *auth.TokenManager, stateStore authstate.Store, loginRateLimit service.LoginRateLimit) http.Handler
	newServer         func(addr string, handler http.Handler, cfg config.HttpServerConfig) appServer
	notify            func(c chan<- os.Signal, sig ...os.Signal)
	shutdownTimeout   time.Duration
}

func defaultAppDeps() appDeps {
	return appDeps{
		loadEnv:    config.LoadEnv,
		loadConfig: config.Load,
		initDB:     database.InitDB,
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
		setupRouter: func(db *gorm.DB, logger *slog.Logger, tokenManager *auth.TokenManager, stateStore authstate.Store, loginRateLimit service.LoginRateLimit) http.Handler {
			return router.SetupRouter(db, logger, tokenManager, router.AuthRuntime{
				StateStore:     stateStore,
				LoginRateLimit: loginRateLimit,
			})
		},
		newServer: func(addr string, router http.Handler, cfg config.HttpServerConfig) appServer {
			return &http.Server{
				Addr:              addr,
				Handler:           middleware.TimeoutHandler(router, cfg.Timeout),
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
	if err := run(getDefaultAppDeps()); err != nil {
		fatalf("application failed: %v", err)
	}
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

	r := deps.setupRouter(db, slog, tokenManager, authStateStore, service.LoginRateLimit{
		AccountLimit: cfg.Auth.LoginRateLimit.AccountLimit,
		IPLimit:      cfg.Auth.LoginRateLimit.IPLimit,
		Window:       cfg.Auth.LoginRateLimit.Window,
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)

	server := deps.newServer(
		addr,
		r,
		cfg.HttpServer.Server,
	)

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting:", "addr", addr)
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
