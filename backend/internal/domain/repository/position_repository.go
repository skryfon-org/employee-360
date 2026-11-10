package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// PositionRepository defines data access methods for tenant-scoped positions.
// Every method takes tenantID explicitly (resolved by the handler from the auth
// context); implementations must not read the tenant from context.Context.
// Create, Update, and Delete take actorID to record created_by, updated_by, and deleted_by audit columns.
type PositionRepository interface {
	Create(ctx context.Context, tenantID, actorID uuid.UUID, position *entity.Position) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Position, error)
	// GetByIDForUpdate is GetByID taking a row lock (SELECT ... FOR UPDATE); it must run inside a transaction.
	GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*entity.Position, error)
	// List returns positions ordered by name; isActive nil returns all, otherwise filters on the flag.
	List(ctx context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Position, int64, error)
	Update(ctx context.Context, tenantID, actorID uuid.UUID, position *entity.Position) error
	Delete(ctx context.Context, tenantID, id, actorID uuid.UUID) error
	ExistsByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error)
	IsReferenced(ctx context.Context, tenantID, id uuid.UUID) (bool, error)
}
