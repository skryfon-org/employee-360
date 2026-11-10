package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormHolidayCalendarSettingsRepository implements repository.HolidayCalendarSettingsRepository.
// Every operation is scoped to the tenantID explicitly provided by the caller.
type gormHolidayCalendarSettingsRepository struct {
	db *gorm.DB
}

// NewGormHolidayCalendarSettingsRepository constructs a GORM-backed HolidayCalendarSettingsRepository.
func NewGormHolidayCalendarSettingsRepository(db *gorm.DB) repository.HolidayCalendarSettingsRepository {
	return &gormHolidayCalendarSettingsRepository{db: db}
}

var _ repository.HolidayCalendarSettingsRepository = (*gormHolidayCalendarSettingsRepository)(nil)

// GetByTenantID retrieves calendar settings for the specified tenant.
func (r *gormHolidayCalendarSettingsRepository) GetByTenantID(c context.Context, tenantID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrHolidayCalendarSettingsNotFound
	}

	var settings entity.HolidayCalendarSettings
	err := database.DBFromContext(c, r.db).
		Where("tenant_id = ?", tenantID).
		First(&settings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrHolidayCalendarSettingsNotFound
		}
		return nil, err
	}
	return &settings, nil
}

// Upsert creates or updates calendar settings for the specified tenant.
func (r *gormHolidayCalendarSettingsRepository) Upsert(c context.Context, tenantID, actorID uuid.UUID, settings *entity.HolidayCalendarSettings) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}

	settings.TenantID = tenantID
	settings.UpdatedBy = &actorID
	now := time.Now().UTC()
	if settings.ID == uuid.Nil {
		settings.ID = uuid.New()
	}
	if settings.CreatedAt.IsZero() {
		settings.CreatedAt = now
	}
	if settings.CreatedBy == nil {
		settings.CreatedBy = &actorID
	}
	settings.UpdatedAt = now

	db := database.DBFromContext(c, r.db)
	err := db.
		Clauses(
			clause.OnConflict{
				Columns: []clause.Column{{Name: "tenant_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"default_view",
					"default_year",
					"week_start_day",
					"updated_at",
					"updated_by",
				}),
			},
			clause.Returning{},
		).
		Create(settings).Error
	if err != nil {
		return err
	}

	// Re-read after upsert to ensure settings has the actual stored row's ID and timestamps
	var stored entity.HolidayCalendarSettings
	if err := db.Where("tenant_id = ?", tenantID).First(&stored).Error; err != nil {
		return err
	}
	*settings = stored
	return nil
}
