package main

import (
	"UsersSystem/config"
	"UsersSystem/router"
	"fmt"
	"log"
)

// type Config struct {
// 	Server struct {
// 		Port int `yaml:"port"`
// 	} `yaml:"server`
// }

func main() {
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
