package main

import (
	"fmt"
	"go-user-system/config"
	"go-user-system/internal/utils"
	"go-user-system/pkg/database"
	"go-user-system/pkg/migration"
	"go-user-system/router"
	"log"
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

	if err := migration.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("run migrations failed: %v", err)
	}

	r := router.SetupRouter(db)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Println("server starting at", addr)

	err = r.Run(addr)
	if err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}
