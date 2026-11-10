package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// TenantReader is the read-only tenant lookup port used to verify that a
// tenant referenced by a token is real and usable.
type TenantReader interface {
	// GetByID returns the tenant with the given id, excluding soft-deleted
	// rows (deleted_at IS NULL). It returns domainerrors.ErrTenantNotFound
	// when no such live tenant exists. The is_active flag is NOT filtered
	// here; callers decide how to treat inactive tenants.
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error)
}

// TenantRepository defines the data access methods for the caller's own
// tenant. tenants is the tenant boundary itself (no tenant_id column): the id
// is passed explicitly, sourced from the auth context by the handler. All
// reads exclude soft-deleted rows.
type TenantRepository interface {
	TenantReader
	// LockByID returns the live tenant and takes a row lock (SELECT ... FOR
	// UPDATE) for the rest of the surrounding transaction.
	LockByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error)
	// UpdateName renames the tenant; ErrTenantNotFound if no live row matched.
	UpdateName(ctx context.Context, id uuid.UUID, name string, actorID uuid.UUID, at time.Time) error
}
