package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

const migrationPingTimeout = 10 * time.Second

type MigrationConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func MigrationConfigFromEnv(getenv func(string) string) (MigrationConfig, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	cfg := MigrationConfig{
		Host:     strings.TrimSpace(getenv("DB_HOST")),
		Port:     strings.TrimSpace(getenv("DB_PORT")),
		User:     strings.TrimSpace(getenv("DB_USER")),
		Password: getenv("DB_PASSWORD"),
		Database: strings.TrimSpace(getenv("DB_NAME")),
	}
	var missing []string
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "DB_HOST", value: cfg.Host},
		{name: "DB_PORT", value: cfg.Port},
		{name: "DB_USER", value: cfg.User},
		{name: "DB_PASSWORD", value: cfg.Password},
		{name: "DB_NAME", value: cfg.Database},
	} {
		if field.value == "" {
			missing = append(missing, field.name)
		}
	}
	if len(missing) > 0 {
		return MigrationConfig{}, fmt.Errorf("migration database config missing: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

func OpenMigrationDBFromEnv(ctx context.Context) (*sql.DB, error) {
	cfg, err := MigrationConfigFromEnv(os.Getenv)
	if err != nil {
		return nil, err
	}
	return OpenMigrationDB(ctx, cfg)
}

func OpenMigrationDB(ctx context.Context, cfg MigrationConfig) (*sql.DB, error) {
	if ctx == nil {
		return nil, errors.New("migration database context is required")
	}
	driverConfig := mysqldriver.NewConfig()
	driverConfig.User = cfg.User
	driverConfig.Passwd = cfg.Password
	driverConfig.Net = "tcp"
	driverConfig.Addr = net.JoinHostPort(cfg.Host, cfg.Port)
	driverConfig.DBName = cfg.Database
	driverConfig.ParseTime = true
	driverConfig.MultiStatements = true

	db, err := sql.Open("mysql", driverConfig.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open migration database: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, migrationPingTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping migration database: %w", err)
	}
	return db, nil
}
