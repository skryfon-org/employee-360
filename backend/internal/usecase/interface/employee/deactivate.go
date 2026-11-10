package employee

import (
	"context"

	"github.com/google/uuid"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// DeactivateEmployeeUseCase deactivates a user and revokes their sessions (admin).
type DeactivateEmployeeUseCase interface {
	Execute(ctx context.Context, actor emptypes.Actor, userID uuid.UUID) (*emptypes.EmployeeDetail, error)
}
