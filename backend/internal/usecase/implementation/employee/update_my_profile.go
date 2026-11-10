package employee

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type updateMyProfileUseCase struct {
	users      repository.UserRepository
	orgNames   repository.OrgNameRepository
	profiles   repository.EmployeeProfileRepository
	settings   repository.OrganizationSettingsRepository
	contacts   repository.EmergencyContactRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewUpdateMyProfileUseCase creates a new UpdateMyProfileUseCase.
func NewUpdateMyProfileUseCase(
	users repository.UserRepository,
	orgNames repository.OrgNameRepository,
	profiles repository.EmployeeProfileRepository,
	settings repository.OrganizationSettingsRepository,
	contacts repository.EmergencyContactRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) empuc.UpdateMyProfileUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateMyProfileUseCase{users: users, orgNames: orgNames, profiles: profiles, settings: settings, contacts: contacts, audit: audit, transactor: transactor}
}

func (uc *updateMyProfileUseCase) Execute(ctx context.Context, tenantID, userID uuid.UUID, in emptypes.UpdateMyProfileInput) (*emptypes.MyProfile, error) {
	actor := emptypes.Actor{TenantID: tenantID, UserID: userID}
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	err := uc.transactor.WithinTransaction(ctx, func(tx context.Context) error {
		user, err := loadTarget(tx, uc.users, tenantID, userID)
		if err != nil {
			return err
		}
		nameChanged, err := applyUserNames(user, in.SelfProfileFields)
		if err != nil {
			return err
		}
		p, isNew, err := profileOrNew(tx, uc.profiles, tenantID, userID)
		if err != nil {
			return err
		}
		profileChanged, err := applySelfFields(p, in.SelfProfileFields)
		if err != nil {
			return err
		}
		if len(nameChanged) > 0 {
			user.UpdatedBy = &userID
			user.UpdatedAt = time.Now().UTC()
			if err := uc.users.Update(tx, tenantID, user); err != nil {
				return err
			}
		}
		changed := append(nameChanged, profileChanged...)
		if len(changed) > 0 || isNew {
			if err := saveProfile(tx, uc.profiles, uc.settings, tenantID, userID, p, isNew); err != nil {
				return err
			}
		}
		if _, err := syncCompletion(tx, uc.profiles, uc.contacts, tenantID, userID, userID); err != nil {
			return err
		}
		if len(changed) == 0 {
			return nil
		}
		return record(tx, uc.audit, actor, domainaudit.ActionEmployeeProfileSelfUpdate, userID, map[string]any{"changed_fields": changed})
	})
	if err != nil {
		return nil, err
	}
	return myProfile(ctx, uc.users, uc.orgNames, uc.profiles, uc.contacts, tenantID, userID)
}
