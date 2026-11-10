package employee

import (
	"context"

	"github.com/google/uuid"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// ChangeEmployeeRoleUseCase switches a user between employee and admin (super_admin only).
type ChangeEmployeeRoleUseCase interface {
	Execute(ctx context.Context, actor emptypes.Actor, userID uuid.UUID, role string) (*emptypes.EmployeeDetail, error)
}
