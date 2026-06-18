package main

import (
	"context"
	"errors"
	"fmt"
	"go-user-system/config"
	"go-user-system/internal/auth"
	"go-user-system/pkg/database"
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
	loadEnv         func() error
	loadConfig      func(path string) (*config.Config, error)
	initDB          func(cfg *config.Config) (*gorm.DB, error)
	newTokenManager func(secret string, issuer string, ttl time.Duration) (*auth.TokenManager, error)
	setupRouter     func(db *gorm.DB, logger *slog.Logger, timeout time.Duration, tokenManager *auth.TokenManager) http.Handler
	newServer       func(addr string, handler http.Handler, cfg config.HttpServerConfig) appServer
	notify          func(c chan<- os.Signal, sig ...os.Signal)
	shutdownTimeout time.Duration
}

func defaultAppDeps() appDeps {
	return appDeps{
		loadEnv:    config.LoadEnv,
		loadConfig: config.Load,
		initDB:     database.InitDB,
		newTokenManager: func(secret string, issuer string, ttl time.Duration) (*auth.TokenManager, error) {
			return auth.NewTokenManager(secret, issuer, ttl)
		},
		setupRouter: func(db *gorm.DB, logger *slog.Logger, timeout time.Duration, tokenManager *auth.TokenManager) http.Handler {
			return router.SetupRouter(db, logger, timeout, tokenManager)
		},
		newServer: func(addr string, handler http.Handler, cfg config.HttpServerConfig) appServer {
			return &http.Server{
				Addr:              addr,
				Handler:           handler,
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
		os.Getenv("JWT_SECRET"),
		"go-user-system",
		time.Duration(cfg.JWT.ExpireHours)*time.Hour,
	)

	if err != nil {
		return fmt.Errorf("new token manager failed: %w", err)
	}

	r := deps.setupRouter(db, slog, cfg.HttpServer.Server.Timeout, tokenManager)

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
