// Package role implements the role usecases.
package role

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	roleusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/role"
)

// ListAssignableRolesUseCaseImpl implements roleusecase.ListAssignableRolesUseCase.
type ListAssignableRolesUseCaseImpl struct {
	roleRepo repository.RoleRepository
}

var _ roleusecase.ListAssignableRolesUseCase = (*ListAssignableRolesUseCaseImpl)(nil)

// NewListAssignableRolesUseCase constructs a ListAssignableRolesUseCaseImpl.
func NewListAssignableRolesUseCase(roleRepo repository.RoleRepository) *ListAssignableRolesUseCaseImpl {
	return &ListAssignableRolesUseCaseImpl{roleRepo: roleRepo}
}

// Execute returns the caller's tenant roles excluding super_admin, which the
// invite flow rejects. tenantID is resolved by the handler from the authenticated
// request; admin authorization is enforced by the route-level RequireRole.
func (u *ListAssignableRolesUseCaseImpl) Execute(c context.Context, actor emptypes.Actor) ([]*entity.Role, error) {
	tenantID := actor.TenantID
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	canGrantAdmin := actor.HasRole(entity.RoleSuperAdmin)
	all, err := u.roleRepo.List(c, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.Role, 0, len(all))
	for _, r := range all {
		// Defence in depth on top of the repository's tenant scoping.
		if r == nil || r.TenantID != tenantID || r.Name == entity.RoleSuperAdmin {
			continue
		}
		// Only a super_admin may grant admin (mirrors the invitation rule).
		if r.Name == entity.RoleAdmin && !canGrantAdmin {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}
