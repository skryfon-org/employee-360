package employee

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

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

// Validation sentinels for the admin employee update. They match
// domainerrors.ErrInvalidEmployeeData and carry the client-facing message.
var (
	ErrFirstNameRequired             = required("first_name")
	ErrFirstNameTooLong              = tooLong("first_name", 100)
	ErrLastNameRequired              = required("last_name")
	ErrLastNameTooLong               = tooLong("last_name", 100)
	ErrEmployeeCodeTooLong           = tooLong("employee_code", 30)
	ErrEmploymentTypeInvalid         = oneOf("employment_type", entity.EmploymentTypes)
	ErrEmploymentStatusRequired      = required("employment_status")
	ErrWorkLocationTooLong           = tooLong("work_location", 150)
	ErrExitDateBeforeJoining         = invalidData("exit_date: must not be before joining_date")
	ErrProbationEndDateBeforeJoining = invalidData("probation_end_date: must not be before joining_date")
	ErrConfirmationDateBeforeJoining = invalidData("confirmation_date: must not be before joining_date")
	ErrChangeReasonTooLong           = tooLong("change_reason", 200)
)

type updateEmployeeUseCase struct {
	users       repository.UserRepository
	profiles    repository.EmployeeProfileRepository
	contacts    repository.EmergencyContactRepository
	assignments repository.UserAssignmentRepository
	directory   repository.EmployeeDirectoryRepository
	orgRefs     repository.OrgReferenceRepository
	settings    repository.OrganizationSettingsRepository
	audit       service.AuditRecorder
	transactor  ucshared.Transactor
}

// NewUpdateEmployeeUseCase creates a new UpdateEmployeeUseCase.
func NewUpdateEmployeeUseCase(
	users repository.UserRepository,
	profiles repository.EmployeeProfileRepository,
	contacts repository.EmergencyContactRepository,
	assignments repository.UserAssignmentRepository,
	directory repository.EmployeeDirectoryRepository,
	orgRefs repository.OrgReferenceRepository,
	settings repository.OrganizationSettingsRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) empuc.UpdateEmployeeUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateEmployeeUseCase{
		users: users, profiles: profiles, contacts: contacts, assignments: assignments,
		directory: directory, orgRefs: orgRefs, settings: settings, audit: audit, transactor: transactor,
	}
}

func (uc *updateEmployeeUseCase) Execute(ctx context.Context, actor emptypes.Actor, userID uuid.UUID, in emptypes.UpdateEmployeeInput) (*emptypes.EmployeeDetail, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	err := uc.transactor.WithinTransaction(ctx, func(tx context.Context) error {
		user, err := loadTarget(tx, uc.users, actor.TenantID, userID)
		if err != nil {
			return err
		}
		if err := guardSuperAdminTarget(actor, user); err != nil {
			return err
		}
		profile, isNew, err := profileOrNew(tx, uc.profiles, actor.TenantID, userID)
		if err != nil {
			return err
		}

		var changed []string

		// --- user fields ---
		nameChanged, err := applyUserNames(user, in.SelfProfileFields)
		if err != nil {
			return err
		}
		changed = append(changed, nameChanged...)

		// --- department / position ---
		org, err := uc.applyOrgAssignment(tx, actor.TenantID, user, in)
		if err != nil {
			return err
		}
		changed = append(changed, org.changedFields()...)

		// --- manager ---
		mgrChanged, err := uc.applyManager(tx, actor.TenantID, userID, profile, in)
		if err != nil {
			return err
		}
		if mgrChanged {
			changed = append(changed, "manager_id")
		}

		// --- employment fields ---
		employmentChanged, err := uc.applyEmployment(tx, actor.TenantID, userID, profile, in)
		if err != nil {
			return err
		}
		changed = append(changed, employmentChanged...)

		// --- personal/address profile fields ---
		selfChanged, err := applySelfFields(profile, in.SelfProfileFields)
		if err != nil {
			return err
		}
		changed = append(changed, selfChanged...)

		if err := checkDateOrder(profile); err != nil {
			return err
		}

		// --- persist ---
		if len(changed) == 0 && !isNew {
			return nil
		}
		user.UpdatedBy = &actor.UserID
		user.UpdatedAt = time.Now().UTC()
		// Applied only now so a failed validation never leaves user half-changed.
		user.DepartmentID, user.PositionID = org.dept, org.pos
		if err := uc.users.Update(tx, actor.TenantID, user); err != nil {
			return err
		}
		if err := saveProfile(tx, uc.profiles, uc.settings, actor.TenantID, actor.UserID, profile, isNew); err != nil {
			return err
		}

		if org.deptChanged || org.posChanged {
			if err := changeAssignment(tx, uc.assignments, uc.audit, actor, profile, org.dept, org.pos, in); err != nil {
				return err
			}
		}
		if _, err := syncCompletion(tx, uc.profiles, uc.contacts, actor.TenantID, userID, actor.UserID); err != nil {
			return err
		}
		return record(tx, uc.audit, actor, domainaudit.ActionEmployeeUpdate, userID, map[string]any{"changed_fields": changed})
	})
	if err != nil {
		return nil, err
	}
	return detail(ctx, uc.directory, uc.profiles, uc.contacts, actor.TenantID, userID)
}

// orgAssignment is the department/position outcome of an update.
type orgAssignment struct {
	dept, pos               *uuid.UUID
	deptChanged, posChanged bool
}

func (a orgAssignment) changedFields() []string {
	var f []string
	if a.deptChanged {
		f = append(f, "department_id")
	}
	if a.posChanged {
		f = append(f, "position_id")
	}
	return f
}

// applyOrgAssignment resolves the new department/position (current values live
// on users) and share-locks + validates any newly assigned one. It does not
// mutate user; the caller writes org.dept/org.pos when persisting.
func (uc *updateEmployeeUseCase) applyOrgAssignment(ctx context.Context, tenantID uuid.UUID, user *entity.User, in emptypes.UpdateEmployeeInput) (orgAssignment, error) {
	org := orgAssignment{dept: user.DepartmentID, pos: user.PositionID}
	if in.DepartmentID.Set {
		org.dept = optUUID(in.DepartmentID)
	}
	if in.PositionID.Set {
		org.pos = optUUID(in.PositionID)
	}
	org.deptChanged = !uuidEq(user.DepartmentID, org.dept)
	org.posChanged = !uuidEq(user.PositionID, org.pos)
	if org.deptChanged && org.dept != nil {
		found, active, lookupErr := uc.orgRefs.LockDepartmentShared(ctx, tenantID, *org.dept)
		if err := requireUsableRef(found, active, lookupErr, domainerrors.ErrDepartmentNotFound, domainerrors.ErrDepartmentInactive); err != nil {
			return org, err
		}
	}
	if org.posChanged && org.pos != nil {
		found, active, lookupErr := uc.orgRefs.LockPositionShared(ctx, tenantID, *org.pos)
		if err := requireUsableRef(found, active, lookupErr, domainerrors.ErrPositionNotFound, domainerrors.ErrPositionInactive); err != nil {
			return org, err
		}
	}
	return org, nil
}

// requireUsableRef maps a lock lookup result to an error: lookup failure, then
// not found, then inactive.
func requireUsableRef(found, active bool, err, notFound, inactive error) error {
	switch {
	case err != nil:
		return err
	case !found:
		return notFound
	case !active:
		return inactive
	}
	return nil
}

// applyManager validates and sets the reporting manager; reports whether it changed.
func (uc *updateEmployeeUseCase) applyManager(ctx context.Context, tenantID, userID uuid.UUID, profile *entity.EmployeeProfile, in emptypes.UpdateEmployeeInput) (bool, error) {
	if !in.ManagerID.Set {
		return false, nil
	}
	newMgr := optUUID(in.ManagerID)
	if uuidEq(profile.ManagerID, newMgr) {
		return false, nil
	}
	if newMgr != nil {
		if *newMgr == userID {
			return false, domainerrors.ErrManagerInvalid
		}
		m, err := uc.users.GetByID(ctx, tenantID, *newMgr)
		if err != nil {
			if errors.Is(err, domainerrors.ErrUserNotFound) {
				return false, domainerrors.ErrManagerInvalid
			}
			return false, err
		}
		if !m.IsActive {
			return false, domainerrors.ErrManagerInvalid
		}
	}
	profile.ManagerID = newMgr
	return true, nil
}

// applyEmployment applies the employment fields (code, type, status, dates,
// work location) to profile and returns the changed field names in order.
func (uc *updateEmployeeUseCase) applyEmployment(ctx context.Context, tenantID, userID uuid.UUID, profile *entity.EmployeeProfile, in emptypes.UpdateEmployeeInput) ([]string, error) {
	var changed []string
	prevCode := profile.EmployeeCode
	if c, err := applyString(in.EmployeeCode, &profile.EmployeeCode, 30, ErrEmployeeCodeTooLong); err != nil {
		return nil, err
	} else if c {
		changed = append(changed, "employee_code")
	}
	if profile.EmployeeCode != nil && !strEq(prevCode, profile.EmployeeCode) {
		if err := uc.checkCodeFormat(ctx, tenantID, *profile.EmployeeCode); err != nil {
			return nil, err
		}
		taken, err := uc.profiles.EmployeeCodeExists(ctx, tenantID, *profile.EmployeeCode, userID)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, domainerrors.ErrEmployeeCodeTaken
		}
	}
	if c, err := applyEnum(in.EmploymentType, &profile.EmploymentType, entity.EmploymentTypes, ErrEmploymentTypeInvalid); err != nil {
		return nil, err
	} else if c {
		changed = append(changed, "employment_type")
	}
	if in.EmploymentStatus.Set {
		v := strings.TrimSpace(in.EmploymentStatus.Value)
		if in.EmploymentStatus.Null || v == "" {
			return nil, ErrEmploymentStatusRequired
		}
		if err := validateEmploymentStatus(v); err != nil {
			return nil, err
		}
		if v != profile.EmploymentStatus {
			profile.EmploymentStatus = v
			changed = append(changed, "employment_status")
		}
	}
	for _, f := range []struct {
		name string
		opt  emptypes.Optional[emptypes.Date]
		dst  **time.Time
	}{
		{"joining_date", in.JoiningDate, &profile.JoiningDate},
		{"probation_end_date", in.ProbationEndDate, &profile.ProbationEndDate},
		{"confirmation_date", in.ConfirmationDate, &profile.ConfirmationDate},
		{"exit_date", in.ExitDate, &profile.ExitDate},
	} {
		if applyDate(f.opt, f.dst) {
			changed = append(changed, f.name)
		}
	}
	if c, err := applyString(in.WorkLocation, &profile.WorkLocation, 150, ErrWorkLocationTooLong); err != nil {
		return nil, err
	} else if c {
		changed = append(changed, "work_location")
	}
	return changed, nil
}

// checkCodeFormat enforces that a manually entered code is the tenant's current
// prefix followed by digits (e.g. EMP0042).
func (uc *updateEmployeeUseCase) checkCodeFormat(ctx context.Context, tenantID uuid.UUID, code string) error {
	cfg, err := uc.settings.GetByTenantID(ctx, tenantID)
	if errors.Is(err, domainerrors.ErrOrganizationSettingsNotFound) {
		cfg, err = entity.DefaultOrganizationSettings(tenantID), nil
	}
	if err != nil {
		return err
	}
	if !entity.EmployeeCodePattern(cfg.EmployeeCodePrefix).MatchString(code) {
		return invalidData(fmt.Sprintf("employee_code: must start with the organization prefix %q followed by digits only (e.g. %s)",
			cfg.EmployeeCodePrefix, entity.FormatEmployeeCode(cfg.EmployeeCodePrefix, cfg.EmployeeCodePadding, cfg.EmployeeCodeNextNumber)))
	}
	return nil
}

func optUUID(o emptypes.Optional[uuid.UUID]) *uuid.UUID {
	if o.Null || o.Value == uuid.Nil {
		return nil
	}
	v := o.Value
	return &v
}

func checkDateOrder(p *entity.EmployeeProfile) error {
	if p.JoiningDate == nil {
		return nil
	}
	for _, f := range []struct {
		d   *time.Time
		err error
	}{
		{p.ExitDate, ErrExitDateBeforeJoining},
		{p.ProbationEndDate, ErrProbationEndDateBeforeJoining},
		{p.ConfirmationDate, ErrConfirmationDateBeforeJoining},
	} {
		if f.d != nil && f.d.Before(*p.JoiningDate) {
			return f.err
		}
	}
	return nil
}

// changeAssignment closes the open stint, opens a new one and audits it. It
// runs inside the caller's transaction, so close + open + users update commit
// atomically.
func changeAssignment(
	ctx context.Context,
	assignments repository.UserAssignmentRepository,
	audit service.AuditRecorder,
	actor emptypes.Actor,
	profile *entity.EmployeeProfile,
	dept, pos *uuid.UUID,
	in emptypes.UpdateEmployeeInput,
) error {
	open, err := assignments.GetOpen(ctx, actor.TenantID, profile.UserID)
	if err != nil {
		return err
	}
	var effective time.Time
	switch {
	case in.EffectiveDate.Set && !in.EffectiveDate.Null:
		effective = utcDate(in.EffectiveDate.Value.Time)
	case open == nil && profile.JoiningDate != nil:
		effective = *profile.JoiningDate
	default:
		effective = utcDate(time.Now())
	}
	if open != nil {
		if effective.Before(open.EffectiveFrom) {
			return domainerrors.ErrAssignmentDateOrder
		}
		if err := assignments.CloseOpen(ctx, actor.TenantID, profile.UserID, actor.UserID, effective); err != nil {
			return err
		}
	}
	var reason *string
	if in.ChangeReason.Set && !in.ChangeReason.Null {
		if v := strings.TrimSpace(in.ChangeReason.Value); v != "" {
			if utf8.RuneCountInString(v) > 200 {
				return ErrChangeReasonTooLong
			}
			reason = &v
		}
	}
	next := &entity.UserAssignment{
		TenantID: actor.TenantID, UserID: profile.UserID, DepartmentID: dept, PositionID: pos,
		EffectiveFrom: effective, ChangeReason: reason,
	}
	if err := assignments.Create(ctx, actor.UserID, next); err != nil {
		return err
	}
	meta := map[string]any{"effective_from": effective.Format(emptypes.DateLayout)}
	if open != nil {
		meta["from_department_id"], meta["from_position_id"] = open.DepartmentID, open.PositionID
	}
	meta["to_department_id"], meta["to_position_id"] = dept, pos
	return record(ctx, audit, actor, domainaudit.ActionEmployeeAssignmentChange, profile.UserID, meta)
}
