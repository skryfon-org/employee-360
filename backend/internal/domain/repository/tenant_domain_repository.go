package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// TenantDomainRepository defines the resolution lookups on tenant domains.
// All reads exclude soft-deleted rows.
type TenantDomainRepository interface {
	// FindTenantByDomain finds the tenant associated with the given domain name (used during login to resolve tenant_id).
	FindTenantByDomain(ctx context.Context, domain string) (*entity.Tenant, error)
	// DomainBelongsToTenant reports whether domain (compared case-insensitively)
	// is a live registered domain of the given active tenant.
	DomainBelongsToTenant(ctx context.Context, tenantID uuid.UUID, domain string) (bool, error)
}

// TenantDomainManager is the super_admin management port over the caller's own
// tenant's domains, kept separate from the narrow login/invitation lookups
// above. Methods take the tenant id explicitly (from the auth context via the
// handler). All reads exclude soft-deleted rows.
type TenantDomainManager interface {
	// Create registers d.Domain (already normalised) for d.TenantID. It returns
	// domainerrors.ErrDomainAlreadyExists when a live row already owns the domain.
	Create(ctx context.Context, d *entity.TenantDomain) error
	// ListByTenantID returns the tenant's live domains, oldest first.
	ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*entity.TenantDomain, error)
	// CountByTenantID counts the tenant's live domains.
	CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error)
	// GetByID returns one live domain of the tenant (ErrDomainNotFound otherwise).
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.TenantDomain, error)
	// UpdateDomain changes the value of the tenant's live domain (already
	// normalised). ErrDomainNotFound when no live row of this tenant matched;
	// ErrDomainAlreadyExists when another live row owns it. A soft-deleted
	// row holding the value is not destroyed: it is renamed to a placeholder.
	UpdateDomain(ctx context.Context, tenantID, id uuid.UUID, domain string, actorID uuid.UUID, at time.Time) error
	// CountUsersOnDomain counts the tenant's non-soft-deleted users (active or
	// not, e.g. invited) whose email domain (the part after the last '@',
	// compared case-insensitively and exactly) equals domain.
	CountUsersOnDomain(ctx context.Context, tenantID uuid.UUID, domain string) (int64, error)
	// SoftDelete marks the tenant's domain deleted (deleted_at/deleted_by);
	// ErrDomainNotFound when no live row matched.
	SoftDelete(ctx context.Context, tenantID, id, actorID uuid.UUID, at time.Time) error
}
