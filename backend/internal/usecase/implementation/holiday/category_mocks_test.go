package holiday

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

type mockHolidayCategoryRepo struct {
	createFn                  func(ctx context.Context, tenantID, actorID uuid.UUID, category *entity.HolidayCategory) error
	updateFn                  func(ctx context.Context, tenantID, actorID uuid.UUID, category *entity.HolidayCategory) error
	softDeleteFn              func(ctx context.Context, tenantID, id, actorID uuid.UUID) error
	getByIDFn                 func(ctx context.Context, tenantID, id uuid.UUID) (*entity.HolidayCategory, error)
	getByIDForUpdateFn        func(ctx context.Context, tenantID, id uuid.UUID) (*entity.HolidayCategory, error)
	listFn                    func(ctx context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.HolidayCategory, int64, error)
	existsByNameFn            func(ctx context.Context, tenantID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error)
	countHolidaysByCategoryFn func(ctx context.Context, tenantID, categoryID uuid.UUID) (int64, error)
}

func (m *mockHolidayCategoryRepo) Create(ctx context.Context, tenantID, actorID uuid.UUID, cat *entity.HolidayCategory) error {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, actorID, cat)
	}
	return nil
}

func (m *mockHolidayCategoryRepo) Update(ctx context.Context, tenantID, actorID uuid.UUID, cat *entity.HolidayCategory) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, actorID, cat)
	}
	return nil
}

func (m *mockHolidayCategoryRepo) SoftDelete(ctx context.Context, tenantID, id, actorID uuid.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, tenantID, id, actorID)
	}
	return nil
}

func (m *mockHolidayCategoryRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.HolidayCategory, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockHolidayCategoryRepo) GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*entity.HolidayCategory, error) {
	if m.getByIDForUpdateFn != nil {
		return m.getByIDForUpdateFn(ctx, tenantID, id)
	}
	return m.GetByID(ctx, tenantID, id)
}

func (m *mockHolidayCategoryRepo) List(ctx context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.HolidayCategory, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, isActive, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockHolidayCategoryRepo) ExistsByName(ctx context.Context, tenantID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(ctx, tenantID, name, excludeID)
	}
	return false, nil
}

func (m *mockHolidayCategoryRepo) CountHolidaysByCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (int64, error) {
	if m.countHolidaysByCategoryFn != nil {
		return m.countHolidaysByCategoryFn(ctx, tenantID, categoryID)
	}
	return 0, nil
}
