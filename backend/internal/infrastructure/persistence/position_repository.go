package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormPositionRepository implements repository.PositionRepository.
// Every operation is scoped to the tenantID explicitly provided by the caller.
type gormPositionRepository struct {
	db *gorm.DB
}

// NewGormPositionRepository constructs a GORM-backed PositionRepository.
func NewGormPositionRepository(db *gorm.DB) repository.PositionRepository {
	return &gormPositionRepository{db: db}
}

var _ repository.PositionRepository = (*gormPositionRepository)(nil)

// scoped returns a DB handle restricted to tenantID and non-deleted rows.
func (r *gormPositionRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
}

// Create persists a new position row scoped to the caller's tenant and records created_by and updated_by.
func (r *gormPositionRepository) Create(c context.Context, tenantID, actorID uuid.UUID, position *entity.Position) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrPositionNotFound
	}
	position.TenantID = tenantID
	position.CreatedBy = &actorID
	position.UpdatedBy = &actorID
	now := time.Now().UTC()
	if position.ID == uuid.Nil {
		position.ID = uuid.New()
	}
	if position.CreatedAt.IsZero() {
		position.CreatedAt = now
	}
	if position.UpdatedAt.IsZero() {
		position.UpdatedAt = now
	}

	err := database.DBFromContext(c, r.db).Create(position).Error
	if err != nil {
		if isUniqueViolation(err) {
			return domainerrors.ErrPositionNameTaken
		}
		return err
	}
	return nil
}

// GetByIDForUpdate looks up a position within the caller's tenant and locks the
// row (FOR UPDATE) until the surrounding transaction ends.
func (r *gormPositionRepository) GetByIDForUpdate(c context.Context, tenantID, id uuid.UUID) (*entity.Position, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrPositionNotFound
	}

	var pos entity.Position
	err := r.scoped(c, tenantID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&pos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrPositionNotFound
		}
		return nil, err
	}
	return &pos, nil
}

// GetByID looks up a position by ID within the caller's tenant.
func (r *gormPositionRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.Position, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrPositionNotFound
	}

	var pos entity.Position
	err := r.scoped(c, tenantID).Where("id = ?", id).First(&pos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrPositionNotFound
		}
		return nil, err
	}
	return &pos, nil
}

// List returns a paginated slice of positions for the caller's tenant.
func (r *gormPositionRepository) List(c context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Position, int64, error) {
	if tenantID == uuid.Nil {
		return []*entity.Position{}, 0, nil
	}

	var (
		positions []*entity.Position
		total     int64
	)

	db := r.scoped(c, tenantID).Model(&entity.Position{})
	if isActive != nil {
		db = db.Where("is_active = ?", *isActive)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		db = db.Limit(limit)
	}
	if offset > 0 {
		db = db.Offset(offset)
	}

	if err := db.Order("name ASC, id ASC").Find(&positions).Error; err != nil {
		return nil, 0, err
	}

	return positions, total, nil
}

// Update modifies position details within the caller's tenant and records updated_by.
func (r *gormPositionRepository) Update(c context.Context, tenantID, actorID uuid.UUID, position *entity.Position) error {
	if tenantID == uuid.Nil || position.ID == uuid.Nil {
		return domainerrors.ErrPositionNotFound
	}

	position.TenantID = tenantID
	position.UpdatedBy = &actorID
	position.UpdatedAt = time.Now().UTC()

	result := r.scoped(c, tenantID).
		Model(&entity.Position{}).
		Where("id = ?", position.ID).
		Updates(map[string]interface{}{
			"name":        position.Name,
			"description": position.Description,
			"is_active":   position.IsActive,
			"updated_at":  position.UpdatedAt,
			"updated_by":  actorID,
		})

	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return domainerrors.ErrPositionNameTaken
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrPositionNotFound
	}

	return nil
}

// Delete soft-deletes a position by ID within the caller's tenant.
// actorID is the admin performing the removal.
func (r *gormPositionRepository) Delete(c context.Context, tenantID, id, actorID uuid.UUID) error {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return domainerrors.ErrPositionNotFound
	}

	now := time.Now().UTC()
	result := database.DBFromContext(c, r.db).
		Model(&entity.Position{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Updates(map[string]any{
			"deleted_at": now,
			"deleted_by": actorID,
			"updated_at": now,
			"updated_by": actorID,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrPositionNotFound
	}
	return nil
}

// ExistsByName checks if another position with the same name exists in the caller's tenant.
func (r *gormPositionRepository) ExistsByName(c context.Context, tenantID uuid.UUID, name string) (bool, error) {
	if tenantID == uuid.Nil {
		return false, nil
	}

	var count int64
	err := r.scoped(c, tenantID).
		Model(&entity.Position{}).
		Where("LOWER(name) = LOWER(?)", strings.TrimSpace(name)).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsReferenced checks whether any active users or pending user_invitations
// (not accepted, revoked or expired) reference this position.
func (r *gormPositionRepository) IsReferenced(c context.Context, tenantID, id uuid.UUID) (bool, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return false, nil
	}

	db := database.DBFromContext(c, r.db)

	var userCount int64
	if err := db.Table("users").
		Where("position_id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Count(&userCount).Error; err != nil {
		return false, err
	}
	if userCount > 0 {
		return true, nil
	}

	var invitationCount int64
	if err := db.Table("user_invitations").
		Where("position_id = ? AND tenant_id = ? AND deleted_at IS NULL AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?",
			id, tenantID, time.Now().UTC()).
		Count(&invitationCount).Error; err != nil {
		return false, err
	}
	return invitationCount > 0, nil
}
