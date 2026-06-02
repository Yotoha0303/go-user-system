package config

import (
	"errors"
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
	DataBase string `yaml:"database"`
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
		return nil, ErrReadConfigFileFailed
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, ErrUnmarshalConfigFileDataFailed
	}

	return &cfg, nil
}
