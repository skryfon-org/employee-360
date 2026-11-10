package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// OrganizationSettingsRepository persists the per-tenant organization_settings
// row. Every method takes the tenantID explicitly (resolved by the handler from
// the auth context) and never reads it from context.Context.
//
// Employee code generation (formatting, skipping taken codes) is usecase logic;
// this port only exposes the atomic counter primitive, NextEmployeeCodeNumber.
type OrganizationSettingsRepository interface {
	// GetByTenantID returns the tenant's row, or
	// domainerrors.ErrOrganizationSettingsNotFound when none exists yet.
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error)
	// EnsureDefaults inserts the default row if the tenant has none (idempotent).
	EnsureDefaults(ctx context.Context, tenantID, actorID uuid.UUID) error
	// LockByTenantID reads the row with SELECT ... FOR UPDATE so it serialises with
	// concurrent code allocation. Must run inside a transaction.
	LockByTenantID(ctx context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error)
	// Update writes the employee code format columns and the counter, scoped by tenant.
	Update(ctx context.Context, tenantID, actorID uuid.UUID, s *entity.OrganizationSettings) error
	// NextEmployeeCodeNumber atomically takes and increments the tenant's employee
	// code counter (single upsert; a tenant without a row gets one with the
	// defaults). It joins the transaction carried by ctx, so a rollback also rolls
	// the counter back. It does not check whether the resulting code is free.
	NextEmployeeCodeNumber(ctx context.Context, tenantID uuid.UUID) (entity.EmployeeCodeCounter, error)
}
