package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	MySQL  MySQLConfig  `yaml:"mysql"`
	JWT    JWTConfig    `yaml:"jwt"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Database string `yaml:"database"`
}

type JWTConfig struct {
	ExpireHours int `yaml:"expireHours"`
}

var (
	ErrReadConfigFileFailed          = errors.New("read file failed")
	ErrUnmarshalConfigFileDataFailed = errors.New("unmarshal config file data failed")
)

func LoadEnv() {
	_ = godotenv.Load()
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReadConfigFileFailed, err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshalConfigFileDataFailed, err)
	}

	return &cfg, nil
}
