// Package tenant holds request/response DTOs for the super_admin current-tenant
// settings usecases. Requests never carry the actor or the tenant; both come
// from the auth context.
package tenant

import (
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// MaxNameLen is the longest accepted tenant name (in characters).
const MaxNameLen = 200

// UpdateTenantRequest renames a tenant.
type UpdateTenantRequest struct {
	Name string `json:"name" example:"Acme Corporation"`
}

// AddDomainRequest registers an additional email domain for a tenant.
type AddDomainRequest struct {
	// Fully qualified domain with at least two labels, e.g. example.com; single-label hosts like localhost are not supported.
	Domain string `json:"domain" example:"acme.io"`
}

// UpdateDomainRequest changes the value of an existing tenant domain.
type UpdateDomainRequest struct {
	// Fully qualified domain with at least two labels, e.g. example.com; single-label hosts like localhost are not supported.
	Domain string `json:"domain" example:"acme.io"`
}

// TenantDetail is a tenant together with its live domains.
type TenantDetail struct {
	Tenant  *entity.Tenant
	Domains []*entity.TenantDomain
}

// TenantResponse is the API representation of a tenant. Audit actor columns
// and soft-delete markers are never exposed.
type TenantResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TenantDomainResponse is the API representation of a tenant domain.
type TenantDomainResponse struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Domain    string    `json:"domain"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantDetailResponse is a tenant with its domains.
type TenantDetailResponse struct {
	TenantResponse
	Domains []TenantDomainResponse `json:"domains"`
}

// ToTenantResponse maps a tenant entity to its API representation.
func ToTenantResponse(t *entity.Tenant) TenantResponse {
	return TenantResponse{ID: t.ID, Name: t.Name, IsActive: t.IsActive, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}

// ToTenantDomainResponse maps a tenant domain entity to its API representation.
func ToTenantDomainResponse(d *entity.TenantDomain) TenantDomainResponse {
	return TenantDomainResponse{ID: d.ID, TenantID: d.TenantID, Domain: d.Domain, CreatedAt: d.CreatedAt}
}

// ToTenantDomainResponses maps a slice (never nil, so JSON renders []).
func ToTenantDomainResponses(ds []*entity.TenantDomain) []TenantDomainResponse {
	out := make([]TenantDomainResponse, 0, len(ds))
	for _, d := range ds {
		out = append(out, ToTenantDomainResponse(d))
	}
	return out
}

// ToTenantDetailResponse maps a detail result to its API representation.
func ToTenantDetailResponse(d *TenantDetail) TenantDetailResponse {
	return TenantDetailResponse{TenantResponse: ToTenantResponse(d.Tenant), Domains: ToTenantDomainResponses(d.Domains)}
}
