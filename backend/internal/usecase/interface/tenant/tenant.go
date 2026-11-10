// Package tenant declares the super_admin current-tenant settings usecase
// ports. The product is self-hosted (one organisation per deployment): there is
// no tenant creation or cross-tenant management. tenantID and actorID are
// resolved by the handler from the auth context, never from the URL or body.
// Usecases never read identity from context.Context.
package tenant

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	tenanttypes "github.com/skryfon-org/employee-360/backend/internal/types/tenant"
)

// GetTenantUseCase returns the caller's tenant with its live domains.
type GetTenantUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID) (*tenanttypes.TenantDetail, error)
}

// RenameTenantUseCase renames the caller's tenant.
type RenameTenantUseCase interface {
	Execute(ctx context.Context, actorID, tenantID uuid.UUID, req tenanttypes.UpdateTenantRequest) (*entity.Tenant, error)
}

// ListTenantDomainsUseCase lists a tenant's live domains.
type ListTenantDomainsUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID) ([]*entity.TenantDomain, error)
}

// AddTenantDomainUseCase registers an extra domain for the tenant.
type AddTenantDomainUseCase interface {
	Execute(ctx context.Context, actorID, tenantID uuid.UUID, req tenanttypes.AddDomainRequest) (*entity.TenantDomain, error)
}

// UpdateTenantDomainUseCase changes the value of one of the tenant's domains.
type UpdateTenantDomainUseCase interface {
	Execute(ctx context.Context, actorID, tenantID, domainID uuid.UUID, req tenanttypes.UpdateDomainRequest) (*entity.TenantDomain, error)
}

// RemoveTenantDomainUseCase soft-deletes a tenant domain (never the last one).
type RemoveTenantDomainUseCase interface {
	Execute(ctx context.Context, actorID, tenantID, domainID uuid.UUID) error
}
