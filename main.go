package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// type Config struct {
// 	Server struct {
// 		Port int `yaml:"port"`
// 	} `yaml:"server`
// }

type Config struct {
	Server ServerConfig `yaml:"server"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {
	cfg, err := loadConfig("config.yml")
	if err != nil {
		panic(fmt.Sprintf("load config failed: %v", err))
	}

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, Response{
			Code: 0,
			Msg:  "success",
			Data: gin.H{
				"message": "pong",
			},
		})
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Println("server starting at", addr)

	err = r.Run(addr)
	if err != nil {
		panic(fmt.Sprintf("server run failed: %v", err))
	}
}
