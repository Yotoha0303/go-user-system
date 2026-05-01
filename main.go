package main

import (
	"fmt"
	"go-user-system/config"
	"go-user-system/dao"
	"go-user-system/global"
	"go-user-system/model"
	"go-user-system/router"
	"go-user-system/utils"
	"log"
)

// type Config struct {
// 	Server struct {
// 		Port int `yaml:"port"`
// 	} `yaml:"server`
// }

func main() {
	if err := utils.InitJWTKey(); err != nil {
		log.Fatalf("init jwt key failed: %v", err)
	}
	db, err := dao.InitDB()

	if err != nil {
		log.Fatalf("failed to connect database")
	}

	if err := db.AutoMigrate(&model.User{}); err != nil {
		log.Fatalf("auto migrate failed:%v", err)
	}
	global.DB = db

	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	r := router.SetupRouter()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Println("server starting at", addr)

	err = r.Run(addr)
	if err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}
