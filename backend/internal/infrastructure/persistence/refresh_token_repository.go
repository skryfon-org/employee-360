package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
)

// gormRefreshTokenRepository is a GORM-backed adapter implementing
// repository.RefreshTokenRepository.
type gormRefreshTokenRepository struct {
	db *gorm.DB
}

// NewGormRefreshTokenRepository constructs a GORM-backed RefreshTokenRepository.
func NewGormRefreshTokenRepository(db *gorm.DB) repository.RefreshTokenRepository {
	return &gormRefreshTokenRepository{db: db}
}

var _ repository.RefreshTokenRepository = (*gormRefreshTokenRepository)(nil)

// Create persists a new refresh token record. ip_address is a Postgres INET
// column (nullable, no default): an empty Go string is not a valid inet
// literal, so an empty IPAddress is omitted from the insert entirely rather
// than written as "", leaving the column NULL.
func (r *gormRefreshTokenRepository) Create(c context.Context, token *entity.RefreshToken) error {
	tx := database.DBFromContext(c, r.db)
	if token.IPAddress == "" {
		tx = tx.Omit("ip_address")
	}
	return tx.Create(token).Error
}

// GetByTokenHash looks up a refresh token by its hash alone. The hash is a
// unique, high-entropy value (uq_refresh_tokens_token_hash), so no further
// scoping is required or possible before authentication has resolved a
// tenant/user -- this is the existing pattern already used by
// TokenRefreshUseCase and LogoutUseCase.
func (r *gormRefreshTokenRepository) GetByTokenHash(c context.Context, tokenHash string) (*entity.RefreshToken, error) {
	var rt entity.RefreshToken
	if err := database.DBFromContext(c, r.db).
		Where("token_hash = ?", tokenHash).
		First(&rt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &rt, nil
}

// Revoke marks a single refresh token as revoked. The update is conditional
// (revoked_at IS NULL) so concurrent rotations of the same token cannot both
// succeed; the loser gets ErrTokenAlreadyRevoked.
func (r *gormRefreshTokenRepository) Revoke(c context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	res := database.DBFromContext(c, r.db).
		Model(&entity.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrTokenAlreadyRevoked
	}
	return nil
}

// RevokeFamily marks every token in a rotation family as revoked (used on
// refresh-token reuse detection).
func (r *gormRefreshTokenRepository) RevokeFamily(c context.Context, family uuid.UUID) error {
	now := time.Now().UTC()
	return database.DBFromContext(c, r.db).
		Model(&entity.RefreshToken{}).
		Where("family = ?", family).
		Update("revoked_at", now).Error
}

// RevokeAllForUser marks every refresh token belonging to a user as revoked
// (used on password reset to invalidate all active sessions).
func (r *gormRefreshTokenRepository) RevokeAllForUser(c context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	return database.DBFromContext(c, r.db).
		Model(&entity.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("revoked_at", now).Error
}

// DeleteExpiredTokens hard-deletes refresh tokens that expired before the
// given time (housekeeping).
func (r *gormRefreshTokenRepository) DeleteExpiredTokens(c context.Context, before time.Time) error {
	return database.DBFromContext(c, r.db).
		Where("expires_at < ?", before).
		Delete(&entity.RefreshToken{}).Error
}
