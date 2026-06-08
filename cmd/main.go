package main

import (
	"context"
	"errors"
	"fmt"
	"go-user-system/config"
	"go-user-system/internal/utils"
	"go-user-system/pkg/database"
	"go-user-system/pkg/migration"
	"go-user-system/router"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type appServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

type appDeps struct {
	loadEnv         func()
	loadConfig      func(path string) (*config.Config, error)
	initJWTKey      func(cfg *config.Config) error
	initDB          func(cfg *config.Config) (*gorm.DB, error)
	runMigrations   func(db *gorm.DB, dir string) error
	setupRouter     func(db *gorm.DB) http.Handler
	newServer       func(addr string, handler http.Handler) appServer
	notify          func(c chan<- os.Signal, sig ...os.Signal)
	shutdownTimeout time.Duration
}

func defaultAppDeps() appDeps {
	return appDeps{
		loadEnv:       config.LoadEnv,
		loadConfig:    config.Load,
		initJWTKey:    utils.InitJWTKey,
		initDB:        database.InitDB,
		runMigrations: migration.RunMigrations,
		setupRouter: func(db *gorm.DB) http.Handler {
			return router.SetupRouter(db)
		},
		newServer: func(addr string, handler http.Handler) appServer {
			return &http.Server{
				Addr:    addr,
				Handler: handler,
			}
		},
		notify:          signal.Notify,
		shutdownTimeout: 10 * time.Second,
	}
}

var getDefaultAppDeps = defaultAppDeps
var fatalf = log.Fatalf

func main() {
	if err := run(getDefaultAppDeps()); err != nil {
		fatalf("application failed: %v", err)
	}
}

func run(deps appDeps) error {
	deps.loadEnv()

	cfg, err := deps.loadConfig("config.yml")
	if err != nil {
		return fmt.Errorf("load config failed: %w", err)
	}

	if err := deps.initJWTKey(cfg); err != nil {
		return fmt.Errorf("init jwt key failed: %w", err)
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

	if err := deps.runMigrations(db, "migrations"); err != nil {
		return fmt.Errorf("run migrations failed: %w", err)
	}

	r := deps.setupRouter(db)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := deps.newServer(addr, r)

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("server starting: addr=%s", addr)
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

	log.Printf("server shutting down")
	if deps.shutdownTimeout == 0 {
		deps.shutdownTimeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), deps.shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}
	log.Printf("server stopped")
	return nil
}
