package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormHolidayRepository implements repository.HolidayRepository.
// Every operation is scoped to the tenantID explicitly provided by the caller.
type gormHolidayRepository struct {
	db *gorm.DB
}

// NewGormHolidayRepository constructs a GORM-backed HolidayRepository.
func NewGormHolidayRepository(db *gorm.DB) repository.HolidayRepository {
	return &gormHolidayRepository{db: db}
}

var _ repository.HolidayRepository = (*gormHolidayRepository)(nil)

// scoped returns a DB handle restricted to tenantID and non-deleted rows.
func (r *gormHolidayRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
}

func isHolidayUniqueViolation(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// normalizeHolidayDate returns date truncated to UTC midnight.
func normalizeHolidayDate(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

// Create persists a new holiday row scoped to the caller's tenant and records created_by and updated_by.
func (r *gormHolidayRepository) Create(c context.Context, tenantID, actorID uuid.UUID, holiday *entity.Holiday) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}

	holiday.TenantID = tenantID
	holiday.CreatedBy = &actorID
	holiday.UpdatedBy = &actorID
	now := time.Now().UTC()
	if holiday.ID == uuid.Nil {
		holiday.ID = uuid.New()
	}
	if holiday.CreatedAt.IsZero() {
		holiday.CreatedAt = now
	}
	if holiday.UpdatedAt.IsZero() {
		holiday.UpdatedAt = now
	}
	holiday.Date = normalizeHolidayDate(holiday.Date)

	err := database.DBFromContext(c, r.db).Create(holiday).Error
	if err != nil {
		if isHolidayUniqueViolation(err) {
			return domainerrors.ErrHolidayDuplicate
		}
		return err
	}
	return nil
}

// Update modifies holiday details within the caller's tenant and records updated_by.
func (r *gormHolidayRepository) Update(c context.Context, tenantID, actorID uuid.UUID, holiday *entity.Holiday) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	if holiday.ID == uuid.Nil {
		return domainerrors.ErrHolidayNotFound
	}

	holiday.TenantID = tenantID
	holiday.UpdatedBy = &actorID
	holiday.UpdatedAt = time.Now().UTC()
	holiday.Date = normalizeHolidayDate(holiday.Date)

	result := r.scoped(c, tenantID).
		Model(&entity.Holiday{}).
		Where("id = ?", holiday.ID).
		Updates(map[string]interface{}{
			"category_id": holiday.CategoryID,
			"name":        holiday.Name,
			"description": holiday.Description,
			"date":        holiday.Date,
			"updated_at":  holiday.UpdatedAt,
			"updated_by":  actorID,
		})

	if result.Error != nil {
		if isHolidayUniqueViolation(result.Error) {
			return domainerrors.ErrHolidayDuplicate
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrHolidayNotFound
	}

	return nil
}

// SoftDelete marks a holiday as deleted within the caller's tenant, recording deleted_at and deleted_by.
func (r *gormHolidayRepository) SoftDelete(c context.Context, tenantID, id, actorID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	if id == uuid.Nil {
		return domainerrors.ErrHolidayNotFound
	}

	now := time.Now().UTC()
	result := database.DBFromContext(c, r.db).
		Model(&entity.Holiday{}).
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
		return domainerrors.ErrHolidayNotFound
	}
	return nil
}

// GetByID looks up a holiday by ID within the caller's tenant.
func (r *gormHolidayRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.Holiday, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrHolidayNotFound
	}

	var holiday entity.Holiday
	err := withCategoryFlag(r.scoped(c, tenantID)).Where("id = ?", id).First(&holiday).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrHolidayNotFound
		}
		return nil, err
	}
	return &holiday, nil
}

// ListByDateRange returns the tenant's holidays with from <= date < to
// (inclusive from, exclusive to, matching ListByYear), ordered by date ASC, name ASC.
func (r *gormHolidayRepository) ListByDateRange(c context.Context, tenantID uuid.UUID, from, to time.Time) ([]*entity.Holiday, error) {
	if tenantID == uuid.Nil {
		return []*entity.Holiday{}, nil
	}
	holidays := []*entity.Holiday{}
	if err := withCategoryFlag(r.scoped(c, tenantID)).Where("date >= ? AND date < ?", from, to).
		Order("date ASC, name ASC").Find(&holidays).Error; err != nil {
		return nil, err
	}
	return holidays, nil
}

// ListByYear queries holidays within the year range [year-01-01, year+1-01-01) for the tenant.
// If month is provided, it narrows the query range to that month. Results are ordered by date ASC, name ASC.
func (r *gormHolidayRepository) ListByYear(c context.Context, tenantID uuid.UUID, year int, categoryID *uuid.UUID, month *int) ([]*entity.Holiday, error) {
	if tenantID == uuid.Nil {
		return []*entity.Holiday{}, nil
	}

	var startDate, endDate time.Time
	if month != nil {
		startDate = time.Date(year, time.Month(*month), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0)
	} else {
		startDate = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	db := withCategoryFlag(r.scoped(c, tenantID)).
		Where("date >= ? AND date < ?", startDate, endDate)

	if categoryID != nil && *categoryID != uuid.Nil {
		db = db.Where("category_id = ?", *categoryID)
	}

	var holidays []*entity.Holiday
	if err := db.Order("date ASC, name ASC").Find(&holidays).Error; err != nil {
		return nil, err
	}

	return holidays, nil
}

// ExistsDuplicate checks if another non-deleted holiday exists in the tenant with the same date and name (case-insensitive).
// If excludeID is provided, that holiday ID is ignored (used for updates).
func (r *gormHolidayRepository) ExistsDuplicate(c context.Context, tenantID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error) {
	if tenantID == uuid.Nil {
		return false, nil
	}

	normDate := normalizeHolidayDate(date)
	db := r.scoped(c, tenantID).
		Model(&entity.Holiday{}).
		Where("date = ? AND LOWER(name) = LOWER(?)", normDate, strings.TrimSpace(name))

	if excludeID != nil && *excludeID != uuid.Nil {
		db = db.Where("id != ?", *excludeID)
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// withCategoryFlag selects holidays.* plus the category's is_public flag as
// category_is_public. The category is matched on the same tenant_id.
func withCategoryFlag(db *gorm.DB) *gorm.DB {
	return db.Select(`holidays.*, COALESCE((SELECT hc.is_public FROM holiday_categories hc
		WHERE hc.id = holidays.category_id AND hc.tenant_id = holidays.tenant_id), FALSE) AS category_is_public`)
}
