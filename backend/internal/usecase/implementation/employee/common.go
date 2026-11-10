// Package employee implements employee management and employee self-service
// profile usecases. All repository calls take the explicit tenantID resolved
// by the handler from the auth context; nothing here reads identity from ctx.
package employee

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
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
)

// invalidData is a client-correctable validation failure. Its text is the
// client-facing "field: message" string; it matches
// domainerrors.ErrInvalidEmployeeData so the HTTP layer maps it to 400.
type invalidData string

func (e invalidData) Error() string { return string(e) }

// Is lets callers match domainerrors.ErrInvalidEmployeeData.
func (e invalidData) Is(target error) bool { return target == domainerrors.ErrInvalidEmployeeData }

func tooLong(field string, max int) error {
	return invalidData(fmt.Sprintf("%s: must not exceed %d characters", field, max))
}

func required(field string) error { return invalidData(field + ": is required") }

func oneOf(field string, allowed []string) error {
	return invalidData(fmt.Sprintf("%s: must be one of %s", field, strings.Join(allowed, ", ")))
}

// Validation sentinels shared by the admin update and the self-service profile
// update (the personal and address fields).
var (
	ErrGenderInvalid        = oneOf("gender", entity.Genders)
	ErrBloodGroupInvalid    = oneOf("blood_group", entity.BloodGroups)
	ErrMaritalStatusInvalid = oneOf("marital_status", entity.MaritalStatuses)
	ErrDateOfBirthInvalid   = invalidData("date_of_birth: must be a past date after 1900")
	ErrNationalityTooLong   = tooLong("nationality", 100)
	ErrPhoneTooLong         = tooLong("phone", 30)
	ErrPhoneInvalid         = invalidData("phone: is not a valid phone number")
	ErrPersonalEmailTooLong = tooLong("personal_email", 200)
	ErrPersonalEmailInvalid = invalidData("personal_email: is not a valid email address")
	ErrAddressLine1TooLong  = tooLong("address_line1", 255)
	ErrAddressLine2TooLong  = tooLong("address_line2", 255)
	ErrCityTooLong          = tooLong("city", 100)
	ErrStateTooLong         = tooLong("state", 100)
	ErrCountryTooLong       = tooLong("country", 100)
	ErrPincodeTooLong       = tooLong("pincode", 20)

	// ErrEmploymentStatusInvalid is returned (wrapped with the offending
	// value) for an unknown employment status, in updates and list filters.
	ErrEmploymentStatusInvalid = invalidData("employment_status: invalid value")
)

func requireActor(a emptypes.Actor) error {
	if a.TenantID == uuid.Nil || a.UserID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	return nil
}

func hasRole(roles []entity.Role, name string) bool {
	for _, r := range roles {
		if r.Name == name {
			return true
		}
	}
	return false
}

func roleNames(roles []entity.Role) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, r.Name)
	}
	return out
}

// guardSuperAdminTarget enforces "admin cannot act on super_admin".
func guardSuperAdminTarget(actor emptypes.Actor, target *entity.User) error {
	if hasRole(target.Roles, entity.RoleSuperAdmin) && !actor.HasRole(entity.RoleSuperAdmin) {
		return domainerrors.ErrCannotActOnSuperAdmin
	}
	return nil
}

// ---- field application helpers ----

var (
	phoneRe = regexp.MustCompile(`^[0-9+()\-\s.]{5,30}$`)
)

func strEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func timeEq(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func uuidEq(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// applyString applies an optional text value (trimmed; empty or null clears)
// and reports whether the stored value changed. errTooLong is returned when the
// value exceeds maxLen characters.
func applyString(o emptypes.Optional[string], dst **string, maxLen int, errTooLong error) (bool, error) {
	if !o.Set {
		return false, nil
	}
	var next *string
	if !o.Null {
		if v := strings.TrimSpace(o.Value); v != "" {
			if utf8.RuneCountInString(v) > maxLen {
				return false, errTooLong
			}
			next = &v
		}
	}
	if strEq(*dst, next) {
		return false, nil
	}
	*dst = next
	return true, nil
}

// applyEnum is applyString restricted to the allowed values; errInvalid is
// returned for anything else.
func applyEnum(o emptypes.Optional[string], dst **string, allowed []string, errInvalid error) (bool, error) {
	if o.Set && !o.Null {
		v := strings.TrimSpace(o.Value)
		if v != "" {
			ok := false
			for _, a := range allowed {
				if a == v {
					ok = true
					break
				}
			}
			if !ok {
				return false, errInvalid
			}
			o.Value = v
		}
	}
	return applyString(o, dst, 50, errInvalid)
}

func applyDate(o emptypes.Optional[emptypes.Date], dst **time.Time) bool {
	if !o.Set {
		return false
	}
	var next *time.Time
	if !o.Null {
		t := o.Value.Time.UTC()
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		next = &t
	}
	if timeEq(*dst, next) {
		return false
	}
	*dst = next
	return true
}

func utcDate(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// validateEmploymentStatus rejects values outside entity.EmploymentStatuses.
func validateEmploymentStatus(v string) error {
	for _, a := range entity.EmploymentStatuses {
		if a == v {
			return nil
		}
	}
	return fmt.Errorf("%w %q", ErrEmploymentStatusInvalid, v)
}

// applyUserNames applies the optional first/last name overrides to the users
// row (trimmed, non-empty, max 100 chars) and returns the changed field names.
// The caller persists the user.
func applyUserNames(u *entity.User, f emptypes.SelfProfileFields) ([]string, error) {
	var changed []string
	for _, n := range []struct {
		name        string
		opt         emptypes.Optional[string]
		dst         *string
		errRequired error
		errTooLong  error
	}{
		{"first_name", f.FirstName, &u.FirstName, ErrFirstNameRequired, ErrFirstNameTooLong},
		{"last_name", f.LastName, &u.LastName, ErrLastNameRequired, ErrLastNameTooLong},
	} {
		if !n.opt.Set {
			continue
		}
		v := strings.TrimSpace(n.opt.Value)
		if n.opt.Null || v == "" {
			return nil, n.errRequired
		}
		if utf8.RuneCountInString(v) > 100 {
			return nil, n.errTooLong
		}
		if v != *n.dst {
			*n.dst = v
			changed = append(changed, n.name)
		}
	}
	return changed, nil
}

// applySelfFields applies the personal + address fields shared by the admin
// update and the employee's own PATCH /me/profile, returning changed field names.
func applySelfFields(p *entity.EmployeeProfile, f emptypes.SelfProfileFields) ([]string, error) {
	var changed []string
	track := func(name string, c bool, err error) error {
		if err != nil {
			return err
		}
		if c {
			changed = append(changed, name)
		}
		return nil
	}
	c, err := applyEnum(f.Gender, &p.Gender, entity.Genders, ErrGenderInvalid)
	if err := track("gender", c, err); err != nil {
		return nil, err
	}
	if applyDate(f.DateOfBirth, &p.DateOfBirth) {
		if p.DateOfBirth != nil && (p.DateOfBirth.After(time.Now().UTC()) || p.DateOfBirth.Year() < 1900) {
			return nil, ErrDateOfBirthInvalid
		}
		changed = append(changed, "date_of_birth")
	}
	c, err = applyEnum(f.BloodGroup, &p.BloodGroup, entity.BloodGroups, ErrBloodGroupInvalid)
	if err := track("blood_group", c, err); err != nil {
		return nil, err
	}
	c, err = applyEnum(f.MaritalStatus, &p.MaritalStatus, entity.MaritalStatuses, ErrMaritalStatusInvalid)
	if err := track("marital_status", c, err); err != nil {
		return nil, err
	}
	c, err = applyString(f.Nationality, &p.Nationality, 100, ErrNationalityTooLong)
	if err := track("nationality", c, err); err != nil {
		return nil, err
	}
	c, err = applyString(f.Phone, &p.Phone, 30, ErrPhoneTooLong)
	if err := track("phone", c, err); err != nil {
		return nil, err
	}
	if p.Phone != nil && !phoneRe.MatchString(*p.Phone) {
		return nil, ErrPhoneInvalid
	}
	c, err = applyString(f.PersonalEmail, &p.PersonalEmail, 200, ErrPersonalEmailTooLong)
	if err := track("personal_email", c, err); err != nil {
		return nil, err
	}
	if p.PersonalEmail != nil {
		if a, perr := mail.ParseAddress(*p.PersonalEmail); perr != nil || a.Address != *p.PersonalEmail {
			return nil, ErrPersonalEmailInvalid
		}
	}
	for _, s := range []struct {
		name   string
		opt    emptypes.Optional[string]
		dst    **string
		max    int
		errLen error
	}{
		{"address_line1", f.AddressLine1, &p.AddressLine1, 255, ErrAddressLine1TooLong},
		{"address_line2", f.AddressLine2, &p.AddressLine2, 255, ErrAddressLine2TooLong},
		{"city", f.City, &p.City, 100, ErrCityTooLong},
		{"state", f.State, &p.State, 100, ErrStateTooLong},
		{"country", f.Country, &p.Country, 100, ErrCountryTooLong},
		{"pincode", f.Pincode, &p.Pincode, 20, ErrPincodeTooLong},
	} {
		c, err := applyString(s.opt, s.dst, s.max, s.errLen)
		if err := track(s.name, c, err); err != nil {
			return nil, err
		}
	}
	return changed, nil
}

// ---- shared data access ----

func loadTarget(ctx context.Context, users repository.UserRepository, tenantID, userID uuid.UUID) (*entity.User, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUserNotFound
	}
	return users.GetByIDWithRoles(ctx, tenantID, userID)
}

// authorizeTarget loads the target user and applies the super_admin guard when
// the actor is acting on someone else's data. Self-service (actor == target)
// skips the guard.
func authorizeTarget(ctx context.Context, users repository.UserRepository, actor emptypes.Actor, userID uuid.UUID) error {
	target, err := loadTarget(ctx, users, actor.TenantID, userID)
	if err != nil {
		return err
	}
	if actor.UserID == userID {
		return nil
	}
	return guardSuperAdminTarget(actor, target)
}

// profileOrNew loads the profile, or returns a fresh unsaved one.
func profileOrNew(ctx context.Context, profiles repository.EmployeeProfileRepository, tenantID, userID uuid.UUID) (p *entity.EmployeeProfile, isNew bool, err error) {
	p, err = profiles.GetByUserID(ctx, tenantID, userID)
	if err == nil {
		return p, false, nil
	}
	if errors.Is(err, domainerrors.ErrEmployeeProfileNotFound) {
		return &entity.EmployeeProfile{
			ID: uuid.New(), TenantID: tenantID, UserID: userID,
			EmploymentStatus: entity.EmploymentStatusActive,
		}, true, nil
	}
	return nil, false, err
}

// maxCodeAllocationAttempts bounds how many counter values are skipped because a
// manually assigned code already occupies them.
const maxCodeAllocationAttempts = 100

// ErrEmployeeCodeExhausted is returned when no free employee code was found
// within maxCodeAllocationAttempts counter values.
var ErrEmployeeCodeExhausted = errors.New("could not allocate a free employee code")

// allocateEmployeeCode takes the tenant's next free employee code. It draws counter
// values from settings.NextEmployeeCodeNumber (atomic) and skips any whose formatted
// code is already held by a live profile (a manual override). It must run inside the
// same transaction as the profile insert so a rollback also rolls the counter back.
func allocateEmployeeCode(ctx context.Context, settings repository.OrganizationSettingsRepository, profiles repository.EmployeeProfileRepository, tenantID uuid.UUID) (string, error) {
	for i := 0; i < maxCodeAllocationAttempts; i++ {
		counter, err := settings.NextEmployeeCodeNumber(ctx, tenantID)
		if err != nil {
			return "", err
		}
		code := counter.Code()
		// uuid.Nil excludes no real user: the profile being created has no row yet.
		taken, err := profiles.EmployeeCodeExists(ctx, tenantID, code, uuid.Nil)
		if err != nil {
			return "", err
		}
		if !taken {
			return code, nil
		}
	}
	return "", ErrEmployeeCodeExhausted
}

// saveProfile creates (isNew) or updates the profile. A new profile without a
// manually set code gets the tenant's next generated one first. Must be called
// inside a transaction.
func saveProfile(ctx context.Context, profiles repository.EmployeeProfileRepository, settings repository.OrganizationSettingsRepository, tenantID, actorID uuid.UUID, p *entity.EmployeeProfile, isNew bool) error {
	if isNew {
		if p.EmployeeCode == nil {
			code, err := allocateEmployeeCode(ctx, settings, profiles, tenantID)
			if err != nil {
				return err
			}
			p.EmployeeCode = &code
		}
		return profiles.Create(ctx, actorID, p)
	}
	return profiles.Update(ctx, tenantID, actorID, p)
}

// completion evaluates the mandatory fields + at least one emergency contact.
func completion(ctx context.Context, contacts repository.EmergencyContactRepository, tenantID, userID uuid.UUID, p *entity.EmployeeProfile) (missing []string, err error) {
	missing = p.MissingRequiredFields()
	n, err := contacts.CountByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		missing = append(missing, "emergency_contact")
	}
	return missing, nil
}

// syncCompletion recomputes profile_completed_at: set when everything mandatory
// is present, cleared when it stops being. Persists only on change. Must be
// called after the mutation, inside the same transaction.
func syncCompletion(ctx context.Context, profiles repository.EmployeeProfileRepository, contacts repository.EmergencyContactRepository, tenantID, userID, actorID uuid.UUID) (completed bool, err error) {
	p, err := profiles.GetByUserID(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrEmployeeProfileNotFound) {
			return false, nil
		}
		return false, err
	}
	missing, err := completion(ctx, contacts, tenantID, userID, p)
	if err != nil {
		return false, err
	}
	complete := len(missing) == 0
	switch {
	case complete && p.ProfileCompletedAt == nil:
		now := time.Now().UTC()
		p.ProfileCompletedAt = &now
	case !complete && p.ProfileCompletedAt != nil:
		p.ProfileCompletedAt = nil
	default:
		return complete, nil
	}
	return complete, profiles.Update(ctx, tenantID, actorID, p)
}

func detail(
	ctx context.Context,
	directory repository.EmployeeDirectoryRepository,
	profiles repository.EmployeeProfileRepository,
	contacts repository.EmergencyContactRepository,
	tenantID, userID uuid.UUID,
) (*emptypes.EmployeeDetail, error) {
	sum, err := directory.GetSummary(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	p, err := profiles.GetByUserID(ctx, tenantID, userID)
	if err != nil {
		if !errors.Is(err, domainerrors.ErrEmployeeProfileNotFound) {
			return nil, err
		}
		p = nil
	}
	cs, err := contacts.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	return &emptypes.EmployeeDetail{Summary: sum, Profile: p, Contacts: cs}, nil
}

// myProfile builds the caller's profile view with completion state.
func myProfile(ctx context.Context, users repository.UserRepository, orgNames repository.OrgNameRepository, profiles repository.EmployeeProfileRepository, contacts repository.EmergencyContactRepository, tenantID, userID uuid.UUID) (*emptypes.MyProfile, error) {
	p, err := profiles.GetByUserID(ctx, tenantID, userID)
	if err != nil {
		if !errors.Is(err, domainerrors.ErrEmployeeProfileNotFound) {
			return nil, err
		}
		p = nil
	}
	cs, err := contacts.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	missing := p.MissingRequiredFields()
	if len(cs) == 0 {
		missing = append(missing, "emergency_contact")
	}
	out := &emptypes.MyProfile{Profile: p, Contacts: cs, Completed: len(missing) == 0, MissingFields: missing}
	u, err := loadTarget(ctx, users, tenantID, userID)
	if err != nil {
		return nil, err
	}
	out.FirstName, out.LastName = u.FirstName, u.LastName
	if u.DepartmentID != nil {
		name, found, err := orgNames.DepartmentName(ctx, tenantID, *u.DepartmentID)
		if err != nil {
			return nil, err
		}
		if found {
			out.DepartmentID, out.DepartmentName = u.DepartmentID, name
		}
	}
	if u.PositionID != nil {
		name, found, err := orgNames.PositionName(ctx, tenantID, *u.PositionID)
		if err != nil {
			return nil, err
		}
		if found {
			out.PositionID, out.PositionName = u.PositionID, name
		}
	}
	return out, nil
}

func record(ctx context.Context, audit service.AuditRecorder, actor emptypes.Actor, action string, target uuid.UUID, meta map[string]any) error {
	return audit.Record(ctx, actor.TenantID, actor.UserID, action, domainaudit.EntityEmployee, target, meta)
}

// othersHoldingRole reports whether any active user other than excludeID holds
// the role. The matching rows are locked FOR UPDATE so concurrent demotions /
// deactivations of the last two admins serialise.
func othersHoldingRole(ctx context.Context, directory repository.EmployeeDirectoryRepository, tenantID uuid.UUID, role string, excludeID uuid.UUID) (bool, error) {
	ids, err := directory.LockActiveUserIDsWithRole(ctx, tenantID, role)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id != excludeID {
			return true, nil
		}
	}
	return false, nil
}
