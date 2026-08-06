package repository

import (
	"context"
	"go-user-system/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, db *gorm.DB, token *model.RefreshToken) error
	FindByJTIForUpdate(ctx context.Context, db *gorm.DB, jti string) (*model.RefreshToken, error)
	RevokeByJTI(ctx context.Context, db *gorm.DB, jti string, revokedAt time.Time, replacedByJTI *string) error
	RevokeAllByUserID(ctx context.Context, db *gorm.DB, userID int64, revokedAt time.Time) error
}

type GormRefreshTokenRepository struct{}

func NewGormRefreshTokenRepository() GormRefreshTokenRepository {
	return GormRefreshTokenRepository{}
}

func (GormRefreshTokenRepository) Create(ctx context.Context, db *gorm.DB, token *model.RefreshToken) error {
	return withContext(ctx, db).Create(token).Error
}

func (GormRefreshTokenRepository) FindByJTIForUpdate(ctx context.Context, db *gorm.DB, jti string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	err := withContext(ctx, db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("jti = ?", jti).
		First(&token).
		Error
	return &token, err
}

func (GormRefreshTokenRepository) RevokeByJTI(ctx context.Context, db *gorm.DB, jti string, revokedAt time.Time, replacedByJTI *string) error {
	return withContext(ctx, db).
		Model(&model.RefreshToken{}).
		Where("jti = ? AND revoked_at IS NULL", jti).
		Updates(map[string]interface{}{
			"revoked_at":      revokedAt,
			"replaced_by_jti": replacedByJTI,
		}).
		Error
}

func (GormRefreshTokenRepository) RevokeAllByUserID(ctx context.Context, db *gorm.DB, userID int64, revokedAt time.Time) error {
	return withContext(ctx, db).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", revokedAt).
		Error
}

func withContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return db.WithContext(ctx)
}
