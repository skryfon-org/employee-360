package position

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

type mockPositionRepo struct {
	createFn       func(ctx context.Context, tenantID, actorID uuid.UUID, position *entity.Position) error
	getByIDFn      func(ctx context.Context, tenantID, id uuid.UUID) (*entity.Position, error)
	listFn         func(ctx context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Position, int64, error)
	updateFn       func(ctx context.Context, tenantID, actorID uuid.UUID, position *entity.Position) error
	deleteFn       func(ctx context.Context, tenantID, id, actorID uuid.UUID) error
	existsByNameFn func(ctx context.Context, tenantID uuid.UUID, name string) (bool, error)
	forUpdateCalls int
	isReferencedFn func(ctx context.Context, tenantID, id uuid.UUID) (bool, error)
}

func (m *mockPositionRepo) Create(ctx context.Context, tenantID, actorID uuid.UUID, p *entity.Position) error {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, actorID, p)
	}
	return nil
}

func (m *mockPositionRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Position, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockPositionRepo) GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*entity.Position, error) {
	m.forUpdateCalls++
	return m.GetByID(ctx, tenantID, id)
}

func (m *mockPositionRepo) List(ctx context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Position, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, isActive, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockPositionRepo) Update(ctx context.Context, tenantID, actorID uuid.UUID, p *entity.Position) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, actorID, p)
	}
	return nil
}

func (m *mockPositionRepo) Delete(ctx context.Context, tenantID, id, actorID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, id, actorID)
	}
	return nil
}

func (m *mockPositionRepo) ExistsByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(ctx, tenantID, name)
	}
	return false, nil
}

func (m *mockPositionRepo) IsReferenced(ctx context.Context, tenantID, id uuid.UUID) (bool, error) {
	if m.isReferencedFn != nil {
		return m.isReferencedFn(ctx, tenantID, id)
	}
	return false, nil
}
