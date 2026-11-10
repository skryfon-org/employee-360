package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// UserRepository defines the data access methods for users.
//
// Every method that needs tenant scoping takes tenantID as an explicit
// parameter and never reads it from context.Context -- the repository layer
// has zero knowledge of how tenant_id was resolved (auth middleware read by
// the handler, a token row, a domain lookup, ...). Callers (usecases) are responsible for
// obtaining a trustworthy tenantID and passing it in; Create is the one
// exception, since it already receives a full *entity.User with TenantID set
// by the caller.
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.User, error)
	GetByTenantAndEmail(ctx context.Context, tenantID uuid.UUID, email string) (*entity.User, error)
	GetByIDWithRoles(ctx context.Context, tenantID, id uuid.UUID) (*entity.User, error)
	GetByTenantAndEmailWithRoles(ctx context.Context, tenantID uuid.UUID, email string) (*entity.User, error)
	Update(ctx context.Context, tenantID uuid.UUID, user *entity.User) error
	// Delete soft-deletes the user; actorID is recorded as deleted_by.
	Delete(ctx context.Context, tenantID, id, actorID uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*entity.User, int64, error)
}
