package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config failed: %v", err)
	}
	return path
}

func TestLoadReadsConfigFile(t *testing.T) {
	path := writeTempConfig(t, `
server:
  port: 8082
mysql:
  host: 127.0.0.1
  port: "3306"
  user: root
  database: go_user_system
jwt:
  expireHours: 24
`)

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
}

func TestLoadAppliesEnvOverrides(t *testing.T) {
	t.Setenv("APP_PORT", "9090")
	t.Setenv("DB_HOST", "mysql")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("DB_USER", "app_user")
	t.Setenv("DB_NAME", "app_db")
	t.Setenv("JWT_EXPIRE_HOURS", "12")

	path := writeTempConfig(t, `
server:
  port: 8082
mysql:
  host: 127.0.0.1
  port: "3306"
  user: root
  database: go_user_system
jwt:
  expireHours: 24
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("expected server port 9090, got %d", cfg.Server.Port)
	}
	if cfg.MySQL.Host != "mysql" {
		t.Fatalf("expected mysql host mysql, got %s", cfg.MySQL.Host)
	}
	if cfg.MySQL.Port != "3307" {
		t.Fatalf("expected mysql port 3307, got %s", cfg.MySQL.Port)
	}
	if cfg.MySQL.User != "app_user" {
		t.Fatalf("expected mysql user app_user, got %s", cfg.MySQL.User)
	}
	if cfg.MySQL.Database != "app_db" {
		t.Fatalf("expected mysql database app_db, got %s", cfg.MySQL.Database)
	}
	if cfg.JWT.ExpireHours != 12 {
		t.Fatalf("expected jwt expire hours 12, got %d", cfg.JWT.ExpireHours)
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
	if err := os.WriteFile(path, []byte(`
server:
  port: 8082
mysql:
  host: 127.0.0.1
  port: "3306"
  user: root
  database: go_user_system
jwt:
  expireHours: 24
`), 0o600); err != nil {
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
