// Package employee declares the usecase contracts for employee management and
// the employee's own profile / first-login flow.
package employee

import (
	"context"

	"github.com/google/uuid"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// ListEmployeesUseCase lists tenant employees (admin).
type ListEmployeesUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, q emptypes.ListEmployeesQuery) (*emptypes.ListEmployeesResult, error)
}
