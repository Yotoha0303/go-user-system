package model

import "time"

const (
	RefreshTokenRevokedReasonRotated        = "rotated"
	RefreshTokenRevokedReasonLogout         = "logout"
	RefreshTokenRevokedReasonPasswordChange = "password_change"
	RefreshTokenRevokedReasonReplay         = "replay_detected"
)

type RefreshToken struct {
	ID            int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        int64      `gorm:"not null;index:idx_refresh_tokens_user_id" json:"user_id"`
	JTI           string     `gorm:"size:64;not null;uniqueIndex:uk_refresh_tokens_jti" json:"jti"`
	FamilyID      string     `gorm:"size:64;not null;index:idx_refresh_tokens_family_id" json:"family_id"`
	TokenHash     string     `gorm:"size:64;not null;uniqueIndex:uk_refresh_tokens_hash" json:"-"`
	ExpiresAt     time.Time  `gorm:"not null;index:idx_refresh_tokens_expires_at" json:"expires_at"`
	RevokedAt     *time.Time `gorm:"column:revoked_at" json:"revoked_at,omitempty"`
	RevokedReason *string    `gorm:"size:32;column:revoked_reason" json:"revoked_reason,omitempty"`
	ReplacedByJTI *string    `gorm:"size:64;column:replaced_by_jti" json:"replaced_by_jti,omitempty"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
