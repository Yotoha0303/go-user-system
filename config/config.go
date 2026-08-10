package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Environment string       `yaml:"environment"`
	Server      ServerConfig `yaml:"server"`
	MySQL       MySQLConfig  `yaml:"mysql"`
	Redis       RedisConfig  `yaml:"redis"`
	JWT         JWTConfig    `yaml:"jwt"`
	Auth        AuthConfig   `yaml:"auth"`
	HttpServer  HttpServer   `yaml:"http"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Database string `yaml:"database"`

	MaxOpenConns    int           `yaml:"maxOpenConns"`
	MaxIdleConns    int           `yaml:"maxIdleConns"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifeTime"`
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime"`
	PingTimeout     time.Duration `yaml:"pingTimeout"`
}

type RedisConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Address      string        `yaml:"address"`
	Password     string        `yaml:"-"`
	DB           int           `yaml:"db"`
	DialTimeout  time.Duration `yaml:"dialTimeout"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
	PingTimeout  time.Duration `yaml:"pingTimeout"`
}

type AuthConfig struct {
	Registration   RegistrationConfig   `yaml:"registration"`
	LoginRateLimit LoginRateLimitConfig `yaml:"loginRateLimit"`
	RefreshCookie  RefreshCookieConfig  `yaml:"refreshCookie"`
}

type RefreshCookieConfig struct {
	Secure bool `yaml:"secure"`
}

type RegistrationConfig struct {
	Enabled *bool `yaml:"enabled"`
}

func (c AuthConfig) RegistrationEnabled() bool {
	return c.Registration.Enabled == nil || *c.Registration.Enabled
}

type LoginRateLimitConfig struct {
	AccountLimit int64         `yaml:"accountLimit"`
	IPLimit      int64         `yaml:"ipLimit"`
	Window       time.Duration `yaml:"window"`
}

type JWTConfig struct {
	ExpireHours              int    `yaml:"expireHours"`
	AccessTokenExpireMinutes int    `yaml:"accessTokenExpireMinutes"`
	RefreshTokenExpireHours  int    `yaml:"refreshTokenExpireHours"`
	Algorithm                string `yaml:"algorithm"`
	Secret                   string `yaml:"secret"`
}

type HttpServer struct {
	Server         HttpServerConfig `yaml:"server"`
	TrustedProxies []string         `yaml:"trustedProxies"`
}

type HttpServerConfig struct {
	ReadTimeOut       time.Duration `yaml:"readTimeout"`
	WriteTimeout      time.Duration `yaml:"writeTimeout"`
	IdleTimeout       time.Duration `yaml:"idleTimeout"`
	ReadHeaderTimeout time.Duration `yaml:"readHeaderTimeout"`
	MaxHeaderBytesKib int           `yaml:"maxHeaderBytesKib"`
	Timeout           time.Duration `yaml:"timeout"`
}

var (
	ErrReadConfigFileFailed               = errors.New("read file failed")
	ErrUnmarshalConfigFileDataFailed      = errors.New("unmarshal config file data failed")
	ErrInvalidServerPort                  = errors.New("invalid server port")
	ErrInvalidExpireHours                 = errors.New("invalid expire hours")
	ErrInvalidMySQLPort                   = errors.New("invalid mysql port")
	ErrMySQLDatabaseNotFound              = errors.New("MySQL database name not found")
	ErrMySQLUserNotFound                  = errors.New("MySQL user not found")
	ErrMySQLHostNotFound                  = errors.New("MySQL host not found")
	ErrInvalidHttpServerReadTimeout       = errors.New("invalid server read time out")
	ErrInvalidHttpServerWriteTimeout      = errors.New("invalid server write time out")
	ErrInvalidHttpServerIdleTimeout       = errors.New("invalid server idle time out")
	ErrInvalidHttpServerReadHeaderTimeout = errors.New("invalid server read header time out")
	ErrInvalidHttpServerMaxHeaderBytes    = errors.New("invalid server max header bytes")
	ErrInvalidHttpServerTimeout           = errors.New("invalid http server time out")
	ErrMySQLMaxOpenConnsFailed            = errors.New("MySQL max open conns failed")
	ErrMySQLMaxIdleConnsFailed            = errors.New("MySQL mysql max idle conns failed")
	ErrMySQLInvalidConnMaxIdleTime        = errors.New("invalid mysql conn max idle time")
	ErrMySQLInvalidConnMaxLifetime        = errors.New("invalid mysql conn max life time")
	ErrMySQLInvalidPingTimeout            = errors.New("invalid mysql conn ping time out")
	ErrRedisAddressEmpty                  = errors.New("redis address is empty")
	ErrRedisDBInvalid                     = errors.New("redis db is invalid")
	ErrRedisTimeoutInvalid                = errors.New("redis timeout is invalid")
	ErrLoginRateLimitInvalid              = errors.New("login rate limit is invalid")
	ErrEnvironmentInvalid                 = errors.New("application environment is invalid")
	ErrTrustedProxyInvalid                = errors.New("trusted proxy must be an IP address or CIDR")
	ErrProductionRedisRequired            = errors.New("production environment requires Redis authentication state")
	ErrProductionSecureCookieRequired     = errors.New("production environment requires secure refresh cookies")
)

func (c Config) Validate() error {
	server := c.Server
	mysql := c.MySQL
	jwt := c.JWT
	http := c.HttpServer.Server
	loginRateLimit := c.Auth.LoginRateLimit

	switch c.Environment {
	case "development", "test", "production":
	default:
		return fmt.Errorf("%w: %q", ErrEnvironmentInvalid, c.Environment)
	}

	if server.Port <= 0 {
		return ErrInvalidServerPort
	}

	if jwt.ExpireHours <= 0 {
		return ErrInvalidExpireHours
	}

	if jwt.AccessTokenExpireMinutes <= 0 {
		return ErrInvalidExpireHours
	}

	if jwt.RefreshTokenExpireHours <= 0 {
		return ErrInvalidExpireHours
	}

	if jwt.Algorithm != "HS256" {
		return fmt.Errorf("invalid JWT algorithm: %s (supported: HS256)", jwt.Algorithm)
	}

	if len(jwt.Secret) < 32 {
		return fmt.Errorf("invalid JWT secret: must be at least 32 characters (got %d)", len(jwt.Secret))
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

	if http.Timeout <= 0 {
		return ErrInvalidHttpServerTimeout
	}

	if mysql.MaxOpenConns <= 0 {
		return ErrMySQLMaxOpenConnsFailed
	}

	if mysql.MaxIdleConns < 0 || mysql.MaxIdleConns > mysql.MaxOpenConns {
		return ErrMySQLMaxIdleConnsFailed
	}

	if mysql.ConnMaxIdleTime <= 0 {
		return ErrMySQLInvalidConnMaxIdleTime
	}

	if mysql.ConnMaxLifetime <= 0 {
		return ErrMySQLInvalidConnMaxLifetime
	}

	if mysql.PingTimeout <= 0 {
		return ErrMySQLInvalidPingTimeout
	}

	if c.Redis.Enabled {
		if c.Redis.Address == "" {
			return ErrRedisAddressEmpty
		}
		if c.Redis.DB < 0 {
			return ErrRedisDBInvalid
		}
		if c.Redis.DialTimeout <= 0 || c.Redis.ReadTimeout <= 0 || c.Redis.WriteTimeout <= 0 || c.Redis.PingTimeout <= 0 {
			return ErrRedisTimeoutInvalid
		}
	}

	if loginRateLimit.AccountLimit <= 0 || loginRateLimit.IPLimit <= 0 || loginRateLimit.Window <= 0 {
		return ErrLoginRateLimitInvalid
	}

	for _, trustedProxy := range c.HttpServer.TrustedProxies {
		if net.ParseIP(trustedProxy) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(trustedProxy); err != nil {
			return fmt.Errorf("%w: %q", ErrTrustedProxyInvalid, trustedProxy)
		}
	}

	if c.Environment == "production" {
		if !c.Redis.Enabled {
			return ErrProductionRedisRequired
		}
		if !c.Auth.RefreshCookie.Secure {
			return ErrProductionSecureCookieRequired
		}
	}

	return nil
}

func LoadEnv() error {
	if err := loadEnvFile(".env"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load env file failed: %w", err)
	}
	return nil
}

func loadEnvFile(name string) error {
	if path, ok := findFileUpward(name); ok {
		if err := godotenv.Load(path); err != nil {
			return err
		}
		return nil
	}

	if err := godotenv.Load(); err != nil {
		return err
	}
	return nil
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
		return nil, fmt.Errorf("%w: %w", ErrReadConfigFileFailed, err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnmarshalConfigFileDataFailed, err)
	}

	if err := applyEnvOverrides(&cfg); err != nil {
		return nil, err
	}

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

func applyEnvOverrides(cfg *Config) error {
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.Environment = strings.ToLower(strings.TrimSpace(v))
	}

	if v := os.Getenv("APP_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid APP_PORT: %w", err)
		}
		cfg.Server.Port = port
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

	if v := os.Getenv("REDIS_ENABLED"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid REDIS_ENABLED: %w", err)
		}
		cfg.Redis.Enabled = enabled
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Address = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		db, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid REDIS_DB: %w", err)
		}
		cfg.Redis.DB = db
	}

	if v := os.Getenv("JWT_EXPIRE_HOURS"); v != "" {
		hours, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid JWT_EXPIRE_HOURS: %w", err)
		}
		cfg.JWT.ExpireHours = hours
	}

	if v := os.Getenv("JWT_ACCESS_TOKEN_EXPIRE_MINUTES"); v != "" {
		minutes, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid JWT_ACCESS_TOKEN_EXPIRE_MINUTES: %w", err)
		}
		cfg.JWT.AccessTokenExpireMinutes = minutes
	}

	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}

	if v := os.Getenv("JWT_REFRESH_TOKEN_EXPIRE_HOURS"); v != "" {
		hours, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid JWT_REFRESH_TOKEN_EXPIRE_HOURS: %w", err)
		}
		cfg.JWT.RefreshTokenExpireHours = hours
	}

	if v := os.Getenv("REGISTRATION_ENABLED"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid REGISTRATION_ENABLED: %w", err)
		}
		cfg.Auth.Registration.Enabled = &enabled
	}
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		secure, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid COOKIE_SECURE: %w", err)
		}
		cfg.Auth.RefreshCookie.Secure = secure
	}
	if v := os.Getenv("TRUSTED_PROXIES"); v != "" {
		cfg.HttpServer.TrustedProxies = splitCommaSeparated(v)
	}
	return nil
}

func applyDefaults(cfg *Config) {
	http := &cfg.HttpServer.Server
	jwt := &cfg.JWT
	redis := &cfg.Redis
	loginRateLimit := &cfg.Auth.LoginRateLimit

	if strings.TrimSpace(cfg.Environment) == "" {
		cfg.Environment = "development"
	} else {
		cfg.Environment = strings.ToLower(strings.TrimSpace(cfg.Environment))
	}

	if jwt.AccessTokenExpireMinutes == 0 && jwt.ExpireHours > 0 {
		jwt.AccessTokenExpireMinutes = jwt.ExpireHours * 60
	}
	if jwt.RefreshTokenExpireHours == 0 {
		jwt.RefreshTokenExpireHours = 24 * 7
	}
	if jwt.Algorithm == "" {
		jwt.Algorithm = "HS256"
	}

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

	if redis.Address == "" {
		redis.Address = "127.0.0.1:6379"
	}
	if redis.DialTimeout == 0 {
		redis.DialTimeout = 3 * time.Second
	}
	if redis.ReadTimeout == 0 {
		redis.ReadTimeout = 2 * time.Second
	}
	if redis.WriteTimeout == 0 {
		redis.WriteTimeout = 2 * time.Second
	}
	if redis.PingTimeout == 0 {
		redis.PingTimeout = 3 * time.Second
	}

	if loginRateLimit.AccountLimit == 0 {
		loginRateLimit.AccountLimit = 5
	}
	if loginRateLimit.IPLimit == 0 {
		loginRateLimit.IPLimit = 20
	}
	if loginRateLimit.Window == 0 {
		loginRateLimit.Window = 15 * time.Minute
	}
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
