package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	UserStatusActive   int8 = 1
	UserStatusDisabled int8 = 2
)

type User struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string         `gorm:"size:64;not null;uniqueIndex:idx_username" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Nickname     string         `gorm:"size:64;not null;default:''" json:"nickname"`
	Status       int8           `gorm:"not null;default:1" json:"status"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	LastLoginAt  *time.Time     `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
	DeletedAt    gorm.DeletedAt `gorm:"index:idx_users_deleted_at" json:"-"`
}

func (User) TableName() string {
	return "users"
}
