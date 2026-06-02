package main

import (
	"fmt"
	"go-user-system/config"
	"go-user-system/database"
	"go-user-system/global"
	"go-user-system/model"
	"go-user-system/router"
	"go-user-system/utils"
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

	if err := db.AutoMigrate(&model.User{}); err != nil {
		log.Fatalf("auto migrate failed:%v", err)
	}
	global.DB = db

	r := router.SetupRouter()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Println("server starting at", addr)

	err = r.Run(addr)
	if err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}
