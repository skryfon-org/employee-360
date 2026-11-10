package employee

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

const maxContacts = 10

// Emergency contact validation sentinels (shared by create and update).
var (
	ErrContactNameRequired          = required("name")
	ErrContactRelationshipRequired  = required("relationship")
	ErrContactAlternatePhoneInvalid = invalidData("alternate_phone: is not a valid phone number")
	ErrContactEmailInvalid          = invalidData("email: is not a valid email address")
	ErrEmergencyContactLimitReached = invalidData(fmt.Sprintf("emergency_contacts: at most %d contacts are allowed", maxContacts))
)

type createEmergencyContactUseCase struct {
	users      repository.UserRepository
	profiles   repository.EmployeeProfileRepository
	settings   repository.OrganizationSettingsRepository
	contacts   repository.EmergencyContactRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewCreateEmergencyContactUseCase creates a new CreateEmergencyContactUseCase.
func NewCreateEmergencyContactUseCase(
	users repository.UserRepository,
	profiles repository.EmployeeProfileRepository,
	settings repository.OrganizationSettingsRepository,
	contacts repository.EmergencyContactRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) empuc.CreateEmergencyContactUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &createEmergencyContactUseCase{users: users, profiles: profiles, settings: settings, contacts: contacts, audit: audit, transactor: transactor}
}

func normalizeContact(in emptypes.EmergencyContactInput, c *entity.EmergencyContact) error {
	c.Name = strings.TrimSpace(in.Name)
	c.Relationship = strings.TrimSpace(in.Relationship)
	c.Phone = strings.TrimSpace(in.Phone)
	if c.Name == "" {
		return ErrContactNameRequired
	}
	if c.Relationship == "" {
		return ErrContactRelationshipRequired
	}
	if !phoneRe.MatchString(c.Phone) {
		return ErrPhoneInvalid
	}
	c.AlternatePhone = nil
	if in.AlternatePhone != nil {
		if v := strings.TrimSpace(*in.AlternatePhone); v != "" {
			if !phoneRe.MatchString(v) {
				return ErrContactAlternatePhoneInvalid
			}
			c.AlternatePhone = &v
		}
	}
	c.Email = nil
	if in.Email != nil {
		if v := strings.TrimSpace(*in.Email); v != "" {
			if a, err := mail.ParseAddress(v); err != nil || a.Address != v {
				return ErrContactEmailInvalid
			}
			c.Email = &v
		}
	}
	c.IsPrimary = in.IsPrimary
	return nil
}

func (uc *createEmergencyContactUseCase) Execute(ctx context.Context, actor emptypes.Actor, userID uuid.UUID, in emptypes.EmergencyContactInput) (*entity.EmergencyContact, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	c := &entity.EmergencyContact{ID: uuid.New(), TenantID: actor.TenantID, UserID: userID}
	if err := normalizeContact(in, c); err != nil {
		return nil, err
	}
	err := uc.transactor.WithinTransaction(ctx, func(tx context.Context) error {
		if err := authorizeTarget(tx, uc.users, actor, userID); err != nil {
			return err
		}
		n, err := uc.contacts.CountByUser(tx, actor.TenantID, userID)
		if err != nil {
			return err
		}
		if n >= maxContacts {
			return ErrEmergencyContactLimitReached
		}
		// The first contact becomes primary automatically.
		if n == 0 {
			c.IsPrimary = true
		}
		if c.IsPrimary {
			if err := uc.contacts.ClearPrimary(tx, actor.TenantID, userID, c.ID, actor.UserID); err != nil {
				return err
			}
		}
		if err := uc.contacts.Create(tx, actor.UserID, c); err != nil {
			return err
		}
		if err := ensureProfileRow(tx, uc.profiles, uc.settings, actor, userID); err != nil {
			return err
		}
		if _, err := syncCompletion(tx, uc.profiles, uc.contacts, actor.TenantID, userID, actor.UserID); err != nil {
			return err
		}
		return record(tx, uc.audit, actor, domainaudit.ActionEmployeeContactCreate, userID, map[string]any{"contact_id": c.ID, "is_primary": c.IsPrimary})
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ensureProfileRow creates an empty profile row so completion state has a home.
func ensureProfileRow(ctx context.Context, profiles repository.EmployeeProfileRepository, settings repository.OrganizationSettingsRepository, actor emptypes.Actor, userID uuid.UUID) error {
	p, isNew, err := profileOrNew(ctx, profiles, actor.TenantID, userID)
	if err != nil || !isNew {
		return err
	}
	return saveProfile(ctx, profiles, settings, actor.TenantID, actor.UserID, p, true)
}
