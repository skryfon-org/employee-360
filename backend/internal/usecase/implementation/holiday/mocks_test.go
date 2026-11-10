package holiday

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

type mockHolidayRepo struct {
	createFn          func(ctx context.Context, tenantID, actorID uuid.UUID, holiday *entity.Holiday) error
	updateFn          func(ctx context.Context, tenantID, actorID uuid.UUID, holiday *entity.Holiday) error
	softDeleteFn      func(ctx context.Context, tenantID, id, actorID uuid.UUID) error
	getByIDFn         func(ctx context.Context, tenantID, id uuid.UUID) (*entity.Holiday, error)
	listByYearFn      func(ctx context.Context, tenantID uuid.UUID, year int, categoryID *uuid.UUID, month *int) ([]*entity.Holiday, error)
	existsDuplicateFn func(ctx context.Context, tenantID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error)
}

func (m *mockHolidayRepo) Create(ctx context.Context, tenantID, actorID uuid.UUID, h *entity.Holiday) error {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, actorID, h)
	}
	return nil
}

func (m *mockHolidayRepo) Update(ctx context.Context, tenantID, actorID uuid.UUID, h *entity.Holiday) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, actorID, h)
	}
	return nil
}

func (m *mockHolidayRepo) SoftDelete(ctx context.Context, tenantID, id, actorID uuid.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, tenantID, id, actorID)
	}
	return nil
}

func (m *mockHolidayRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Holiday, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockHolidayRepo) ListByYear(ctx context.Context, tenantID uuid.UUID, year int, categoryID *uuid.UUID, month *int) ([]*entity.Holiday, error) {
	if m.listByYearFn != nil {
		return m.listByYearFn(ctx, tenantID, year, categoryID, month)
	}
	return nil, nil
}

func (m *mockHolidayRepo) ListByDateRange(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]*entity.Holiday, error) {
	return nil, nil
}

func (m *mockHolidayRepo) ExistsDuplicate(ctx context.Context, tenantID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error) {
	if m.existsDuplicateFn != nil {
		return m.existsDuplicateFn(ctx, tenantID, date, name, excludeID)
	}
	return false, nil
}

type mockHolidayCalendarSettingsRepo struct {
	getByTenantIDFn func(ctx context.Context, tenantID uuid.UUID) (*entity.HolidayCalendarSettings, error)
	upsertFn        func(ctx context.Context, tenantID, actorID uuid.UUID, settings *entity.HolidayCalendarSettings) error
}

func (m *mockHolidayCalendarSettingsRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
	if m.getByTenantIDFn != nil {
		return m.getByTenantIDFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockHolidayCalendarSettingsRepo) Upsert(ctx context.Context, tenantID, actorID uuid.UUID, settings *entity.HolidayCalendarSettings) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, tenantID, actorID, settings)
	}
	return nil
}
