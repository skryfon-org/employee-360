package employee

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type getMeUseCase struct {
	users    repository.UserRepository
	profiles repository.EmployeeProfileRepository
	tenants  repository.TenantReader
}

// NewGetMeUseCase creates a new GetMeUseCase.
func NewGetMeUseCase(
	users repository.UserRepository,
	profiles repository.EmployeeProfileRepository,
	tenants repository.TenantReader,
) empuc.GetMeUseCase {
	return &getMeUseCase{users: users, profiles: profiles, tenants: tenants}
}

func isAdminLike(roles []entity.Role) bool {
	return hasRole(roles, entity.RoleAdmin) || hasRole(roles, entity.RoleSuperAdmin)
}

func (uc *getMeUseCase) Execute(ctx context.Context, tenantID, userID uuid.UUID) (*emptypes.Me, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	user, err := loadTarget(ctx, uc.users, tenantID, userID)
	if err != nil {
		return nil, err
	}
	tenant, err := uc.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	me := &emptypes.Me{
		User: user, Roles: roleNames(user.Roles), OrganizationID: tenant.ID, OrganizationName: tenant.Name,
		ProfileRequired: !isAdminLike(user.Roles),
	}
	// A missing profile row simply means onboarding is incomplete; nothing is created here.
	p, err := uc.profiles.GetByUserID(ctx, tenantID, userID)
	switch {
	case err == nil:
		me.ProfileCompleted = p.ProfileCompletedAt != nil
	case errors.Is(err, domainerrors.ErrEmployeeProfileNotFound):
	default:
		return nil, err
	}
	// The welcome screen is shown on every login until onboarding is complete, but only
	// for users subject to profile gating; admins/super_admins always get first_login=false.
	me.FirstLogin = me.ProfileRequired && !me.ProfileCompleted
	return me, nil
}
