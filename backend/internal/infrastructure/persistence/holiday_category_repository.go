package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormHolidayCategoryRepository implements repository.HolidayCategoryRepository.
// Every operation is scoped to the tenantID explicitly provided by the caller.
type gormHolidayCategoryRepository struct {
	db *gorm.DB
}

// NewGormHolidayCategoryRepository constructs a GORM-backed HolidayCategoryRepository.
func NewGormHolidayCategoryRepository(db *gorm.DB) repository.HolidayCategoryRepository {
	return &gormHolidayCategoryRepository{db: db}
}

var _ repository.HolidayCategoryRepository = (*gormHolidayCategoryRepository)(nil)

// scoped returns a DB handle restricted to tenantID and non-deleted rows.
func (r *gormHolidayCategoryRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
}

func isCategoryUniqueViolation(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Create persists a new holiday category row scoped to the caller's tenant.
func (r *gormHolidayCategoryRepository) Create(c context.Context, tenantID, actorID uuid.UUID, category *entity.HolidayCategory) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}

	category.TenantID = tenantID
	category.CreatedBy = &actorID
	category.UpdatedBy = &actorID
	now := time.Now().UTC()
	if category.ID == uuid.Nil {
		category.ID = uuid.New()
	}
	if category.CreatedAt.IsZero() {
		category.CreatedAt = now
	}
	if category.UpdatedAt.IsZero() {
		category.UpdatedAt = now
	}

	err := database.DBFromContext(c, r.db).Create(category).Error
	if err != nil {
		if isCategoryUniqueViolation(err) {
			return domainerrors.ErrHolidayCategoryDuplicate
		}
		return err
	}
	return nil
}

// Update modifies holiday category details within the caller's tenant.
func (r *gormHolidayCategoryRepository) Update(c context.Context, tenantID, actorID uuid.UUID, category *entity.HolidayCategory) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	if category.ID == uuid.Nil {
		return domainerrors.ErrHolidayCategoryNotFound
	}

	category.TenantID = tenantID
	category.UpdatedBy = &actorID
	category.UpdatedAt = time.Now().UTC()

	result := r.scoped(c, tenantID).
		Model(&entity.HolidayCategory{}).
		Where("id = ?", category.ID).
		Updates(map[string]interface{}{
			"name":        category.Name,
			"description": category.Description,
			"color":       category.Color,
			"is_active":   category.IsActive,
			"is_public":   category.IsPublic,
			"updated_at":  category.UpdatedAt,
			"updated_by":  actorID,
		})

	if result.Error != nil {
		if isCategoryUniqueViolation(result.Error) {
			return domainerrors.ErrHolidayCategoryDuplicate
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrHolidayCategoryNotFound
	}

	return nil
}

// SoftDelete marks a holiday category as deleted within the caller's tenant.
func (r *gormHolidayCategoryRepository) SoftDelete(c context.Context, tenantID, id, actorID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	if id == uuid.Nil {
		return domainerrors.ErrHolidayCategoryNotFound
	}

	now := time.Now().UTC()
	result := database.DBFromContext(c, r.db).
		Model(&entity.HolidayCategory{}).
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
		return domainerrors.ErrHolidayCategoryNotFound
	}
	return nil
}

// GetByID looks up a non-deleted holiday category by ID within the caller's tenant.
func (r *gormHolidayCategoryRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.HolidayCategory, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrHolidayCategoryNotFound
	}

	var cat entity.HolidayCategory
	err := r.scoped(c, tenantID).Where("id = ?", id).First(&cat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrHolidayCategoryNotFound
		}
		return nil, err
	}
	return &cat, nil
}

// GetByIDForUpdate looks up a non-deleted holiday category by ID within the caller's tenant
// and locks the row (FOR UPDATE) until the surrounding transaction ends.
func (r *gormHolidayCategoryRepository) GetByIDForUpdate(c context.Context, tenantID, id uuid.UUID) (*entity.HolidayCategory, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrHolidayCategoryNotFound
	}

	var cat entity.HolidayCategory
	err := r.scoped(c, tenantID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&cat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrHolidayCategoryNotFound
		}
		return nil, err
	}
	return &cat, nil
}

// List returns a page of holiday categories for the tenant, optionally filtered
// by is_active, plus the total count matching the filters (before limit/offset).
// Ordering is deterministic: name ASC, id ASC.
func (r *gormHolidayCategoryRepository) List(c context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.HolidayCategory, int64, error) {
	if tenantID == uuid.Nil {
		return []*entity.HolidayCategory{}, 0, nil
	}

	db := r.scoped(c, tenantID).Model(&entity.HolidayCategory{})
	if isActive != nil {
		db = db.Where("is_active = ?", *isActive)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		db = db.Limit(limit)
	}
	if offset > 0 {
		db = db.Offset(offset)
	}

	var categories []*entity.HolidayCategory
	if err := db.Order("name ASC, id ASC").Find(&categories).Error; err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

// ExistsByName checks if another non-deleted category exists in the tenant with the same name (case-insensitive).
func (r *gormHolidayCategoryRepository) ExistsByName(c context.Context, tenantID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	if tenantID == uuid.Nil {
		return false, nil
	}

	db := r.scoped(c, tenantID).
		Model(&entity.HolidayCategory{}).
		Where("LOWER(name) = LOWER(?)", strings.TrimSpace(name))

	if excludeID != nil && *excludeID != uuid.Nil {
		db = db.Where("id != ?", *excludeID)
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountHolidaysByCategory counts the number of non-deleted holidays referencing the category.
func (r *gormHolidayCategoryRepository) CountHolidaysByCategory(c context.Context, tenantID, categoryID uuid.UUID) (int64, error) {
	if tenantID == uuid.Nil || categoryID == uuid.Nil {
		return 0, nil
	}

	var count int64
	err := database.DBFromContext(c, r.db).
		Model(&entity.Holiday{}).
		Where("tenant_id = ? AND category_id = ? AND deleted_at IS NULL", tenantID, categoryID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
