package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config failed: %v", err)
	}
	return path
}

func validConfigYAML() string {
	return `
server:
  port: 8082
mysql:
  host: 127.0.0.1
  port: "3306"
  user: root
  database: go_user_system
  maxOpenConns: 10
  maxIdleConns: 5
  connMaxLifeTime: 30m
  connMaxIdleTime: 5m
  pingTimeout: 3s
jwt:
  expireHours: 24
http:
  server:
    readTimeout: 5s
    writeTimeout: 10s
    idleTimeout: 60s
    readHeaderTimeout: 2s
    maxHeaderBytesKib: 128
    timeout: 5s

`
}

func TestLoadReadsConfigFile(t *testing.T) {
	path := writeTempConfig(t, validConfigYAML())

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if cfg.Server.Port != 8082 {
		t.Fatalf("expected server port 8082, got %d", cfg.Server.Port)
	}
	if cfg.MySQL.Host != "127.0.0.1" {
		t.Fatalf("expected mysql host 127.0.0.1, got %s", cfg.MySQL.Host)
	}
	if cfg.JWT.ExpireHours != 24 {
		t.Fatalf("expected jwt expire hours 24, got %d", cfg.JWT.ExpireHours)
	}

	if cfg.MySQL.MaxOpenConns != 10 {
		t.Fatalf("expected mysql max open conns 10,got %d", cfg.MySQL.MaxOpenConns)
	}
	if cfg.MySQL.MaxIdleConns != 5 {
		t.Fatalf("expected mysql max open conns 5,got %d", cfg.MySQL.MaxIdleConns)
	}
	if cfg.MySQL.ConnMaxLifetime != 30*time.Minute {
		t.Fatalf("expected mysql conn max life time 30m,got %d", cfg.MySQL.ConnMaxLifetime)
	}
	if cfg.MySQL.ConnMaxIdleTime != 5*time.Minute {
		t.Fatalf("expected mysql conn max idle time 5m,got %d", cfg.MySQL.ConnMaxIdleTime)
	}
	if cfg.MySQL.PingTimeout != 3*time.Second {
		t.Fatalf("expected mysql ping time out 3s,got %d", cfg.MySQL.PingTimeout)
	}
}

func validConfig() Config {
	return Config{
		Server: ServerConfig{Port: 8082},
		MySQL: MySQLConfig{
			Host:            "127.0.0.1",
			Port:            "3306",
			User:            "user",
			Database:        "go_user_system",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
			PingTimeout:     3 * time.Second,
		},
		JWT: JWTConfig{ExpireHours: 24},
		HttpServer: HttpServer{
			Server: HttpServerConfig{
				ReadTimeOut:       5 * time.Second,
				WriteTimeout:      10 * time.Second,
				IdleTimeout:       60 * time.Second,
				ReadHeaderTimeout: 2 * time.Second,
				MaxHeaderBytesKib: 128,
				Timeout:           5 * time.Second,
			},
		},
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*Config)
		expectErr error
	}{
		{
			name:      "valid config",
			mutate:    func(cfg *Config) {},
			expectErr: nil,
		},
		{
			name: "invalid server port",
			mutate: func(cfg *Config) {
				cfg.Server.Port = 0
			},
			expectErr: ErrInvalidServerPort,
		},
		{
			name: "invalid jwt expire hours",
			mutate: func(cfg *Config) {
				cfg.JWT.ExpireHours = 0
			},
			expectErr: ErrInvalidExpireHours,
		},
		{
			name: "missing mysql host",
			mutate: func(cfg *Config) {
				cfg.MySQL.Host = ""
			},
			expectErr: ErrMySQLHostNotFound,
		},
		{
			name: "missing mysql port",
			mutate: func(cfg *Config) {
				cfg.MySQL.Port = ""
			},
			expectErr: ErrInvalidMySQLPort,
		},
		{
			name: "non numeric mysql port",
			mutate: func(cfg *Config) {
				cfg.MySQL.Port = "not-a-port"
			},
			expectErr: ErrInvalidMySQLPort,
		},
		{
			name: "mysql port too low",
			mutate: func(cfg *Config) {
				cfg.MySQL.Port = "0"
			},
			expectErr: ErrInvalidMySQLPort,
		},
		{
			name: "mysql port too high",
			mutate: func(cfg *Config) {
				cfg.MySQL.Port = "65536"
			},
			expectErr: ErrInvalidMySQLPort,
		},
		{
			name: "missing mysql database",
			mutate: func(cfg *Config) {
				cfg.MySQL.Database = ""
			},
			expectErr: ErrMySQLDatabaseNotFound,
		},
		{
			name: "missing mysql user",
			mutate: func(cfg *Config) {
				cfg.MySQL.User = ""
			},
			expectErr: ErrMySQLUserNotFound,
		},
		{
			name: "missing http server read time out",
			mutate: func(cfg *Config) {
				cfg.HttpServer.Server.ReadTimeOut = 0
			},
			expectErr: ErrInvalidHttpServerReadTimeout,
		},
		{
			name: "missing http server write time out",
			mutate: func(cfg *Config) {
				cfg.HttpServer.Server.WriteTimeout = 0
			},
			expectErr: ErrInvalidHttpServerWriteTimeout,
		},
		{
			name: "missing http server idle time out",
			mutate: func(cfg *Config) {
				cfg.HttpServer.Server.IdleTimeout = 0
			},
			expectErr: ErrInvalidHttpServerIdleTimeout,
		},
		{
			name: "missing http server read header time out",
			mutate: func(cfg *Config) {
				cfg.HttpServer.Server.ReadHeaderTimeout = 0
			},
			expectErr: ErrInvalidHttpServerReadHeaderTimeout,
		},
		{
			name: "missing http server max header bytes",
			mutate: func(cfg *Config) {
				cfg.HttpServer.Server.MaxHeaderBytesKib = 0
			},
			expectErr: ErrInvalidHttpServerMaxHeaderBytes,
		},
		{
			name: "missing http server time out",
			mutate: func(cfg *Config) {
				cfg.HttpServer.Server.Timeout = 0
			},
			expectErr: ErrInvalidHttpServerTimeout,
		},
		{
			name: "missing mysql max open conns",
			mutate: func(cfg *Config) {
				cfg.MySQL.MaxOpenConns = 0
			},
			expectErr: ErrMySQLMaxOpenConnsFailed,
		},
		{
			name: "missing mysql max idle conns",
			mutate: func(cfg *Config) {
				cfg.MySQL.MaxIdleConns = -1
			},
			expectErr: ErrMySQLMaxIdleConnsFailed,
		},
		{
			name: "missing mysql conn max life time",
			mutate: func(cfg *Config) {
				cfg.MySQL.ConnMaxLifetime = 0
			},
			expectErr: ErrMySQLInvalidConnMaxLifetime,
		},
		{
			name: "missing mysql conn max idle time",
			mutate: func(cfg *Config) {
				cfg.MySQL.ConnMaxIdleTime = 0
			},
			expectErr: ErrMySQLInvalidConnMaxIdleTime,
		},
		{
			name: "missing mysql ping time out",
			mutate: func(cfg *Config) {
				cfg.MySQL.PingTimeout = 0
			},
			expectErr: ErrMySQLInvalidPingTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			err := cfg.Validate()

			if !errors.Is(err, tt.expectErr) {
				t.Fatalf("expected %v, got %v", tt.expectErr, err)
			}
		})
	}
}

func TestLoadReturnsValidationError(t *testing.T) {
	path := writeTempConfig(t, ``)

	_, err := Load(path)

	if !errors.Is(err, ErrInvalidServerPort) {
		t.Fatalf("expected ErrInvalidServerPort, got %v", err)
	}
}

func TestLoadReturnsReadFileError(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yml"))

	if !errors.Is(err, ErrReadConfigFileFailed) {
		t.Fatalf("expected ErrReadConfigFileFailed, got %v", err)
	}
}

func TestLoadReturnsUnmarshalError(t *testing.T) {
	path := writeTempConfig(t, `server: [`)

	_, err := Load(path)

	if !errors.Is(err, ErrUnmarshalConfigFileDataFailed) {
		t.Fatalf("expected ErrUnmarshalConfigFileDataFailed, got %v", err)
	}
}

func TestLoadFindsConfigFromParentDirectory(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "cmd")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatalf("mkdir child failed: %v", err)
	}

	path := filepath.Join(root, "config.yml")
	if err := os.WriteFile(path, []byte(validConfigYAML()), 0o600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore wd failed: %v", err)
		}
	}()

	if err := os.Chdir(child); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	cfg, err := Load("config.yml")
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if cfg.Server.Port != 8082 {
		t.Fatalf("expected server port 8082, got %d", cfg.Server.Port)
	}
}

func TestLoadEnvFindsEnvFromParentDirectory(t *testing.T) {
	const envKey = "GO_USER_SYSTEM_TEST_JWT_SECRET"

	root := t.TempDir()
	child := filepath.Join(root, "cmd")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatalf("mkdir child failed: %v", err)
	}

	envPath := filepath.Join(root, ".env")
	if err := os.WriteFile(envPath, []byte(envKey+"=loaded_from_parent\n"), 0o600); err != nil {
		t.Fatalf("write env failed: %v", err)
	}

	oldValue, hadValue := os.LookupEnv(envKey)
	if err := os.Unsetenv(envKey); err != nil {
		t.Fatalf("unset env failed: %v", err)
	}
	defer func() {
		if hadValue {
			_ = os.Setenv(envKey, oldValue)
		} else {
			_ = os.Unsetenv(envKey)
		}
	}()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore wd failed: %v", err)
		}
	}()

	if err := os.Chdir(child); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	LoadEnv()

	if got := os.Getenv(envKey); got != "loaded_from_parent" {
		t.Fatalf("expected env loaded_from_parent, got %s", got)
	}
}

func TestLoadEnvFileFallsBackWhenFileIsMissing(t *testing.T) {
	loadEnvFile("definitely_missing_env_file_for_test")
}

func TestFindFileUpwardUsesSourceDirectoryWhenWorkingDirectoryIsOutsideProject(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore wd failed: %v", err)
		}
	}()

	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	path, ok := findFileUpward("config.yml")
	if !ok {
		t.Fatal("expected project config.yml to be found")
	}

	if filepath.Base(path) != "config.yml" {
		t.Fatalf("expected config.yml, got %s", path)
	}
}

func TestFindFileUpwardReturnsFalseForMissingRelativeFile(t *testing.T) {
	_, ok := findFileUpward("definitely_missing_config_file_for_test.yml")

	if ok {
		t.Fatal("expected missing file not to be found")
	}
}

func TestSearchStartDirsDeduplicatesWorkingDirectory(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("expected caller file")
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd failed: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore wd failed: %v", err)
		}
	}()

	if err := os.Chdir(filepath.Dir(file)); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	dirs := searchStartDirs()
	seen := make(map[string]struct{})
	for _, dir := range dirs {
		if _, ok := seen[dir]; ok {
			t.Fatalf("expected deduplicated dirs, got duplicate %s in %v", dir, dirs)
		}
		seen[dir] = struct{}{}
	}
}

func TestLoadApplicationsDefaultHTTPServerConfig(t *testing.T) {

	path := writeTempConfig(t, validConfigYAML())

	cfg, err := Load(path)

	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	http := cfg.HttpServer.Server
	if http.ReadTimeOut != 5*time.Second {
		t.Fatalf("expected read timeout 5s, got %s", http.ReadTimeOut)
	}
	if http.WriteTimeout != 10*time.Second {
		t.Fatalf("expected write timeout 10s, got %s", http.WriteTimeout)
	}
}
