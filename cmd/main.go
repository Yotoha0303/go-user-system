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
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	config.LoadEnv()

	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatalf("load config failed: %v", err.Error())
	}

	if err := utils.InitJWTKey(cfg); err != nil {
		log.Fatalf("init jwt key failed: %v", err.Error())
	}

	db, err := database.InitDB(cfg)

	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get database handle failed: %v", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close database failed: %v", err)
		}
	}()

	if err := migration.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("run migrations failed: %v", err)
	}

	r := router.SetupRouter(db)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("server starting: addr=%s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server run failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("server shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Printf("server stopped")
}
