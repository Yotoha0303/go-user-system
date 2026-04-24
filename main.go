package main

import (
	"fmt"
	"go-user-system/config"
	"go-user-system/dao"
	"go-user-system/global"
	"go-user-system/model"
	"go-user-system/router"
	"log"
)

// type Config struct {
// 	Server struct {
// 		Port int `yaml:"port"`
// 	} `yaml:"server`
// }

func main() {
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
		// 程序崩溃，不可修复类型的报错
		// panic(fmt.Sprintf("load config failed: %v", err))
		log.Fatalf("load config failed: %v", err)
	}

	r := router.SetupRouter()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Println("server starting at", addr)

	err = r.Run(addr)
	if err != nil {
		panic(fmt.Sprintf("server run failed: %v", err))
	}
}
