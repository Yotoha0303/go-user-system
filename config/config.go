package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig `yaml:"server"`
	MySQL      MySQLConfig  `yaml:"mysql"`
	JWT        JWTConfig    `yaml:"jwt"`
	HttpServer HttpServer   `yaml:"http"`
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

type HttpServer struct {
	Server HttpServerConfig `yaml:"server"`
}

type HttpServerConfig struct {
	ReadTimeOut       time.Duration `yaml:"readTimeout"`
	WriteTimeout      time.Duration `yaml:"writeTimeout"`
	IdleTimeout       time.Duration `yaml:"idleTimeout"`
	ReadHeaderTimeout time.Duration `yaml:"readHeaderTimeout"`
	MaxHeaderBytesKib int           `yaml:"maxHeaderBytesKib"`
}

var (
	ErrReadConfigFileFailed               = errors.New("read file failed")
	ErrUnmarshalConfigFileDataFailed      = errors.New("unmarshal config file data failed")
	ErrInvalidServerPort                  = errors.New("Invalid server port")
	ErrInvalidExpireHours                 = errors.New("Invalid expire hours")
	ErrInvalidMySQLPort                   = errors.New("invalid mysql port")
	ErrMySQLDatabaseNotFound              = errors.New("MySQL database name not found")
	ErrMySQLUserNotFound                  = errors.New("MySQL user not found")
	ErrMySQLHostNotFound                  = errors.New("MySQL host not found")
	ErrInvalidHttpServerReadTimeout       = errors.New("Invalid server read time out")
	ErrInvalidHttpServerWriteTimeout      = errors.New("Invalid server write time out")
	ErrInvalidHttpServerIdleTimeout       = errors.New("Invalid server idle time out")
	ErrInvalidHttpServerReadHeaderTimeout = errors.New("Invalid server read header time out")
	ErrInvalidHttpServerMaxHeaderBytes    = errors.New("Invalid server max header bytes")
)

func (c Config) Validate() error {
	server := c.Server
	mysql := c.MySQL
	jwt := c.JWT
	http := c.HttpServer.Server

	if server.Port <= 0 {
		return ErrInvalidServerPort
	}

	if jwt.ExpireHours <= 0 {
		return ErrInvalidExpireHours
	}

	if mysql.Host == "" {
		return ErrMySQLHostNotFound
	}

	mysqlPort, err := strconv.Atoi(mysql.Port)
	if err != nil || mysqlPort <= 0 || mysqlPort > 65535 {
		return ErrInvalidMySQLPort
	}

	if mysql.Database == "" {
		return ErrMySQLDatabaseNotFound
	}

	if mysql.User == "" {
		return ErrMySQLUserNotFound
	}

	if http.ReadTimeOut <= 0 {
		return ErrInvalidHttpServerReadTimeout
	}

	if http.WriteTimeout <= 0 {
		return ErrInvalidHttpServerWriteTimeout
	}

	if http.IdleTimeout <= 0 {
		return ErrInvalidHttpServerIdleTimeout
	}

	if http.ReadHeaderTimeout <= 0 {
		return ErrInvalidHttpServerReadHeaderTimeout
	}

	if http.MaxHeaderBytesKib <= 0 {
		return ErrInvalidHttpServerMaxHeaderBytes
	}

	return nil
}

func LoadEnv() {
	loadEnvFile(".env")
}

func loadEnvFile(name string) {
	if path, ok := findFileUpward(name); ok {
		_ = godotenv.Load(path)
		return
	}

	_ = godotenv.Load()
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
	applyDefaults(&cfg)

	if err = cfg.Validate(); err != nil {
		return nil, err
	}

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

func applyDefaults(cfg *Config) {
	http := &cfg.HttpServer.Server

	if http.ReadTimeOut == 0 {
		http.ReadTimeOut = 5 * time.Second
	}
	if http.WriteTimeout == 0 {
		http.WriteTimeout = 10 * time.Second
	}
	if http.IdleTimeout == 0 {
		http.IdleTimeout = 60 * time.Second
	}
	if http.ReadHeaderTimeout == 0 {
		http.ReadHeaderTimeout = 2 * time.Second
	}
	if http.MaxHeaderBytesKib == 0 {
		http.MaxHeaderBytesKib = 512
	}
}
