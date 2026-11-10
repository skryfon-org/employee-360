// Package role declares the role usecase ports.
package role

import (
	"context"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// ListAssignableRolesUseCase lists the caller-tenant roles an admin may assign
// (e.g. when inviting a user). super_admin is never included, and admin is
// included only for a super_admin actor.
type ListAssignableRolesUseCase interface {
	Execute(ctx context.Context, actor emptypes.Actor) ([]*entity.Role, error)
}
