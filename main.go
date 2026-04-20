package main

import (
	"fmt"
	"go-user-system/config"
	"go-user-system/global"
	"go-user-system/model"
	"go-user-system/router"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// type Config struct {
// 	Server struct {
// 		Port int `yaml:"port"`
// 	} `yaml:"server`
// }

func main() {
	dsn := "root:Angel0303@tcp(127.0.0.1:3306)/go_user_system?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

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
