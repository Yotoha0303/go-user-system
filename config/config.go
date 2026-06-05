package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

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
	if path, ok := findFileUpward(".env"); ok {
		_ = godotenv.Load(path)
		return
	}

	err := godotenv.Load()
	if err != nil {
		log.Printf("env failed: %v", err)
	}
}

func Load(path string) (*Config, error) {
	resolvedPath := path
	if _, err := os.Stat(resolvedPath); err != nil {
		if foundPath, ok := findFileUpward(path); ok {
			resolvedPath = foundPath
		}
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReadConfigFileFailed, err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshalConfigFileDataFailed, err)
	}

	applyEnvOverrides(&cfg)

	return &cfg, nil
}

func findFileUpward(name string) (string, bool) {
	if filepath.IsAbs(name) {
		info, err := os.Stat(name)
		return name, err == nil && !info.IsDir()
	}

	for _, dir := range searchStartDirs() {
		if path, ok := findFileUpwardFrom(dir, name); ok {
			return path, true
		}
	}

	return "", false
}

func searchStartDirs() []string {
	dirs := make([]string, 0, 3)
	seen := make(map[string]struct{})

	addDir := func(dir string) {
		if dir == "" {
			return
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return
		}
		if _, ok := seen[absDir]; ok {
			return
		}
		seen[absDir] = struct{}{}
		dirs = append(dirs, absDir)
	}

	dir, err := os.Getwd()
	if err == nil {
		addDir(dir)
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		addDir(filepath.Dir(file))
	}

	if exePath, err := os.Executable(); err == nil {
		addDir(filepath.Dir(exePath))
	}

	return dirs
}

func findFileUpwardFrom(startDir string, name string) (string, bool) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("APP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}

	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.MySQL.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		cfg.MySQL.Port = v
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.MySQL.User = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.MySQL.Database = v
	}

	if v := os.Getenv("JWT_EXPIRE_HOURS"); v != "" {
		if hours, err := strconv.Atoi(v); err == nil {
			cfg.JWT.ExpireHours = hours
		}
	}
}
