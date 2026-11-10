package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// RoleRepository defines the data access methods for roles. Every tenant-scoped
// method takes tenantID explicitly (resolved by the handler from the auth
// context); implementations must not read the tenant from context.Context.
// Create persists role.TenantID, which the usecase sets from its tenantID param.
type RoleRepository interface {
	Create(ctx context.Context, role *entity.Role) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Role, error)
	GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*entity.Role, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*entity.Role, error)
	Update(ctx context.Context, tenantID uuid.UUID, role *entity.Role) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}
