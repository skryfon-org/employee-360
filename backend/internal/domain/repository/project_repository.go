package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// ProjectFilter narrows a project listing. Empty fields are ignored.
type ProjectFilter struct {
	Search string // case-insensitive match on name, code or client name
	Status string
}

// ProjectRepository persists tenant-scoped projects. Every method takes tenantID
// explicitly (resolved by the handler from the auth context). Soft-deleted rows
// are never returned. Create and Update record the actor in the audit columns.
type ProjectRepository interface {
	// Create returns ErrProjectCodeTaken on a duplicate (tenant, code).
	Create(ctx context.Context, tenantID, actorID uuid.UUID, p *entity.Project) error
	// GetByID returns ErrProjectNotFound when missing or in another tenant.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Project, error)
	// GetByIDForUpdate is GetByID with a row lock; it must run in a transaction.
	GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*entity.Project, error)
	// List returns projects ordered by name plus the total matching count.
	List(ctx context.Context, tenantID uuid.UUID, f ProjectFilter, limit, offset int) ([]*entity.Project, int64, error)
	// Update returns ErrProjectNotFound / ErrProjectCodeTaken.
	Update(ctx context.Context, tenantID, actorID uuid.UUID, p *entity.Project) error
	ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error)
}

// ProjectAllocationRepository persists project allocations, tenant scoped and
// soft-delete aware.
type ProjectAllocationRepository interface {
	Create(ctx context.Context, tenantID, actorID uuid.UUID, a *entity.ProjectAllocation) error
	// GetByID returns ErrAllocationNotFound when missing or in another tenant.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.ProjectAllocation, error)
	// ListByProject returns a project's allocations joined with the user's
	// name (tenant-scoped; soft-deleted users still resolve), newest start first,
	// filtered by scope relative to asOf (same date definitions as
	// ListByUserScope; AllocationScopeAll applies no date filter).
	ListByProject(ctx context.Context, tenantID, projectID uuid.UUID, scope entity.AllocationScope, asOf time.Time) ([]*entity.AllocationWithUser, error)
	// ListByUserScope returns the user's non-deleted allocations for the scope
	// relative to asOf (active: start <= asOf
	// and end null or >= asOf; upcoming: start > asOf; past: end < asOf; all: no
	// date filter), joined with project identity, ordered by start_date then project name.
	ListByUserScope(ctx context.Context, tenantID, userID uuid.UUID, scope entity.AllocationScope, asOf time.Time) ([]*entity.AllocationWithProject, error)
	// ListActiveByUser returns the user's non-deleted allocations covering asOf
	// (start <= asOf and end null or >= asOf), joined with project identity,
	// ordered by start_date then project name. Tenant scoped.
	ListActiveByUser(ctx context.Context, tenantID, userID uuid.UUID, asOf time.Time) ([]*entity.AllocationWithProject, error)
	// CountByUserScopes returns the active/upcoming/past counts for the user in
	// one query, with the same definitions as ListByUserScope (active counts
	// only allocations on live projects: active or on_hold).
	CountByUserScopes(ctx context.Context, tenantID, userID uuid.UUID, asOf time.Time) (entity.AllocationScopeCounts, error)
	// HasOverlap reports whether another live allocation of the same user on the
	// same project intersects [start, end] (end nil = open-ended). excludeID is skipped.
	HasOverlap(ctx context.Context, tenantID, projectID, userID uuid.UUID, start time.Time, end *time.Time, excludeID uuid.UUID) (bool, error)
	Update(ctx context.Context, tenantID, actorID uuid.UUID, a *entity.ProjectAllocation) error
	SoftDelete(ctx context.Context, tenantID, id, actorID uuid.UUID) error
}
