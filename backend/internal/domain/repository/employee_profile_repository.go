package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// EmployeeProfileRepository persists employee_profiles. Every method takes the
// tenantID explicitly (sourced from the auth context by the handler) and
// excludes soft-deleted rows.
type EmployeeProfileRepository interface {
	// GetByUserID returns the profile of the user in the tenant, or
	// domainerrors.ErrEmployeeProfileNotFound.
	GetByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*entity.EmployeeProfile, error)
	// Create inserts a profile (profile.TenantID/UserID must be set) as-is; it
	// does not generate an employee code. Callers must assign
	// profile.EmployeeCode first (usecases take it from the organization
	// settings counter inside the same transaction). A nil EmployeeCode is
	// rejected with domainerrors.ErrEmployeeCodeRequired. A duplicate employee
	// code yields domainerrors.ErrEmployeeCodeTaken.
	Create(ctx context.Context, actorID uuid.UUID, profile *entity.EmployeeProfile) error
	// Update writes every column of the profile, scoped by tenant and user.
	Update(ctx context.Context, tenantID, actorID uuid.UUID, profile *entity.EmployeeProfile) error
	// EmployeeCodeExists reports whether another live profile of the tenant
	// already uses the code (excludeUserID is skipped).
	EmployeeCodeExists(ctx context.Context, tenantID uuid.UUID, code string, excludeUserID uuid.UUID) (bool, error)
}
