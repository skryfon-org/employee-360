package employee

import (
	"context"

	"github.com/google/uuid"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// GetEmployeeUseCase returns user + profile + emergency contacts (admin).
type GetEmployeeUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID) (*emptypes.EmployeeDetail, error)
}
