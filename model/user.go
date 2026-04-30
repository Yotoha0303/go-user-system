package model

import "time"

type User struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	Username     string `gorm:"size:64;not null;uniqueIndex"`
	PasswordHash string `gorm:"size:255;not null"`
	Nickname     string `gorm:"size:64;not null;default:''"`
	Status       int8   `gorm:"not null;default:1"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserStatusActive struct {
}
