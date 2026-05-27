package dao

import (
	"fmt"
	"go-user-system/config"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB(cfg *config.Config) (*gorm.DB, error) {

	dbPassword := os.Getenv("DB_PASSWORD")

	if cfg.MySQL.User == "" || dbPassword == "" || cfg.MySQL.Host == "" || cfg.MySQL.Port == "" || cfg.MySQL.DataBase == "" {
		return nil, fmt.Errorf("database config missing")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQL.User,
		dbPassword,
		cfg.MySQL.Host,
		cfg.MySQL.Port,
		cfg.MySQL.DataBase,
	)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
