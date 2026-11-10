package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// EmployeeDirectoryRepository is the read-side port for the admin employee
// directory and for account-level checks that need a cross-table view.
// All methods are tenant-scoped and exclude soft-deleted users.
type EmployeeDirectoryRepository interface {
	// List returns a page of employees matching the filter, newest first, plus
	// the total matching count.
	List(ctx context.Context, tenantID uuid.UUID, f entity.EmployeeFilter, limit, offset int) ([]*entity.EmployeeSummary, int64, error)
	// GetSummary returns one employee, or domainerrors.ErrUserNotFound.
	GetSummary(ctx context.Context, tenantID, userID uuid.UUID) (*entity.EmployeeSummary, error)
	// LockActiveUserIDsWithRole selects (FOR UPDATE) the ids of active,
	// non-deleted users of the tenant holding the named role, serialising
	// concurrent "last admin" decisions. Must run inside a transaction.
	LockActiveUserIDsWithRole(ctx context.Context, tenantID uuid.UUID, roleName string) ([]uuid.UUID, error)
}
