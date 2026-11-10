package employee

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type changeEmployeeRoleUseCase struct {
	users      repository.UserRepository
	userRoles  repository.UserRoleRepository
	roles      repository.RoleRepository
	profiles   repository.EmployeeProfileRepository
	contacts   repository.EmergencyContactRepository
	directory  repository.EmployeeDirectoryRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewChangeEmployeeRoleUseCase creates a new ChangeEmployeeRoleUseCase.
func NewChangeEmployeeRoleUseCase(
	users repository.UserRepository,
	userRoles repository.UserRoleRepository,
	roles repository.RoleRepository,
	profiles repository.EmployeeProfileRepository,
	contacts repository.EmergencyContactRepository,
	directory repository.EmployeeDirectoryRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) empuc.ChangeEmployeeRoleUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &changeEmployeeRoleUseCase{
		users: users, userRoles: userRoles, roles: roles, profiles: profiles, contacts: contacts,
		directory: directory, audit: audit, transactor: transactor,
	}
}

// Execute switches a user between employee and admin. Rules: only super_admin
// may do it; never on oneself; the super_admin role is untouchable; the last
// active admin cannot be demoted.
func (uc *changeEmployeeRoleUseCase) Execute(ctx context.Context, actor emptypes.Actor, userID uuid.UUID, role string) (*emptypes.EmployeeDetail, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	if !actor.HasRole(entity.RoleSuperAdmin) {
		return nil, domainerrors.ErrForbidden
	}
	if userID == actor.UserID {
		return nil, domainerrors.ErrCannotActOnSelf
	}
	if role != entity.RoleAdmin && role != entity.RoleEmployee {
		return nil, domainerrors.ErrRoleChangeNotAllowed
	}
	err := uc.transactor.WithinTransaction(ctx, func(tx context.Context) error {
		user, err := loadTarget(tx, uc.users, actor.TenantID, userID)
		if err != nil {
			return err
		}
		if hasRole(user.Roles, entity.RoleSuperAdmin) {
			return domainerrors.ErrRoleChangeNotAllowed
		}
		wasAdmin := hasRole(user.Roles, entity.RoleAdmin)
		from := entity.RoleEmployee
		if wasAdmin {
			from = entity.RoleAdmin
		}
		if from == role && (wasAdmin || hasRole(user.Roles, entity.RoleEmployee)) {
			return nil // already in the requested role
		}
		if wasAdmin && role == entity.RoleEmployee && user.IsActive {
			ok, err := othersHoldingRole(tx, uc.directory, actor.TenantID, entity.RoleAdmin, userID)
			if err != nil {
				return err
			}
			if !ok {
				return domainerrors.ErrLastActiveAdmin
			}
		}
		target, err := uc.roles.GetByName(tx, actor.TenantID, role)
		if err != nil {
			return err
		}
		if target == nil {
			return domainerrors.ErrRoleNotFound
		}
		for _, r := range user.Roles {
			if r.Name == entity.RoleAdmin || r.Name == entity.RoleEmployee {
				if err := uc.userRoles.RemoveRole(tx, actor.TenantID, userID, r.ID); err != nil {
					return err
				}
			}
		}
		now := time.Now().UTC()
		if err := uc.userRoles.AssignRole(tx, &entity.UserRole{
			ID: uuid.New(), TenantID: actor.TenantID, UserID: userID, RoleID: target.ID,
			CreatedAt: now, UpdatedAt: now, CreatedBy: &actor.UserID, UpdatedBy: &actor.UserID,
		}); err != nil {
			return err
		}
		return record(tx, uc.audit, actor, domainaudit.ActionEmployeeRoleChange, userID, map[string]any{"from": from, "to": role})
	})
	if err != nil {
		return nil, err
	}
	return detail(ctx, uc.directory, uc.profiles, uc.contacts, actor.TenantID, userID)
}
