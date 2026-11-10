package organization

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	orgtypes "github.com/skryfon-org/employee-360/backend/internal/types/organization"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	orguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/organization"
)

// invalidSettings is a client-correctable validation failure ("field: message");
// it matches domainerrors.ErrInvalidOrganizationSettings (HTTP 400).
type invalidSettings string

func (e invalidSettings) Error() string { return string(e) }

func (e invalidSettings) Is(target error) bool {
	return target == domainerrors.ErrInvalidOrganizationSettings
}

var prefixCharsRe = regexp.MustCompile(`^[A-Z0-9]+$`)

// Validation sentinels (exported for tests).
var (
	ErrPrefixRequired        = invalidSettings("employee_code_prefix: is required")
	ErrPrefixTooLong         = invalidSettings(fmt.Sprintf("employee_code_prefix: must not exceed %d characters", entity.MaxEmployeeCodePrefixLen))
	ErrPrefixChars           = invalidSettings("employee_code_prefix: may contain only letters A-Z and digits 0-9 (no separators or spaces)")
	ErrPrefixEndsInDigit     = invalidSettings("employee_code_prefix: must not end in a digit")
	ErrPaddingRange          = invalidSettings(fmt.Sprintf("employee_code_padding: must be between 1 and %d", entity.MaxEmployeeCodePadding))
	ErrStartNumberRange      = invalidSettings(fmt.Sprintf("employee_code_start_number: must be a whole number between 1 and %d", entity.MaxEmployeeCodeStartNumber))
	ErrWelcomeMessageTooLong = invalidSettings(fmt.Sprintf("welcome_message: must not exceed %d characters", entity.MaxWelcomeMessageLen))
	ErrWeeklyOffDaysInvalid  = invalidSettings("weekly_off_days: must contain 1 to 6 unique weekday numbers, each between 0 (Sunday) and 6 (Saturday)")
	ErrCodeTooLong           = invalidSettings(fmt.Sprintf("employee_code_padding: prefix length plus padding must not exceed %d characters", entity.MaxEmployeeCodeLen))
)

// errStartBelowIssued is built per call so the message names the lowest allowed number.
func errStartBelowIssued(min int64) error {
	return invalidSettings(fmt.Sprintf("employee_code_start_number: employee codes have already been issued; the start number must be %d or higher", min))
}

type updateSettingsUseCase struct {
	repo       repository.OrganizationSettingsRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewUpdateSettingsUseCase constructs an UpdateSettingsUseCase.
func NewUpdateSettingsUseCase(repo repository.OrganizationSettingsRepository, audit service.AuditRecorder, transactor ucshared.Transactor) orguc.UpdateSettingsUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateSettingsUseCase{repo: repo, audit: audit, transactor: transactor}
}

// ValidateWeeklyOffDays checks the list is non-empty, unique, each 0..6 and not all
// seven days, and returns it sorted ascending.
func ValidateWeeklyOffDays(days []int) ([]int, error) {
	if len(days) == 0 || len(days) > 6 {
		return nil, ErrWeeklyOffDaysInvalid
	}
	seen := map[int]bool{}
	out := make([]int, 0, len(days))
	for _, d := range days {
		if d < 0 || d > 6 || seen[d] {
			return nil, ErrWeeklyOffDaysInvalid
		}
		seen[d] = true
		out = append(out, d)
	}
	sort.Ints(out)
	return out, nil
}

// ValidateCodeFormat normalizes and validates the format fields (everything that
// does not depend on stored state). It returns the normalized prefix.
func ValidateCodeFormat(rawPrefix string, padding int, start int64) (string, error) {
	prefix := entity.NormalizeEmployeeCodePrefix(rawPrefix)
	switch n := utf8.RuneCountInString(prefix); {
	case n == 0:
		return "", ErrPrefixRequired
	case n > entity.MaxEmployeeCodePrefixLen:
		return "", ErrPrefixTooLong
	}
	if !prefixCharsRe.MatchString(prefix) {
		return "", ErrPrefixChars
	}
	if last := prefix[len(prefix)-1]; last >= '0' && last <= '9' {
		return "", ErrPrefixEndsInDigit
	}
	if padding < 1 || padding > entity.MaxEmployeeCodePadding {
		return "", ErrPaddingRange
	}
	if start < 1 || start > entity.MaxEmployeeCodeStartNumber {
		return "", ErrStartNumberRange
	}
	if len(prefix)+padding > entity.MaxEmployeeCodeLen {
		return "", ErrCodeTooLong
	}
	return prefix, nil
}

func (uc *updateSettingsUseCase) Execute(ctx context.Context, tenantID, actorID uuid.UUID, in orgtypes.UpdateSettingsInput) (*entity.OrganizationSettings, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	prefix, err := ValidateCodeFormat(in.EmployeeCodePrefix, in.EmployeeCodePadding, in.EmployeeCodeStartNumber)
	if err != nil {
		return nil, err
	}

	welcome, setWelcome, err := normalizeWelcomeMessage(in.WelcomeMessage)
	if err != nil {
		return nil, err
	}

	var offDays []int
	if in.WeeklyOffDays != nil {
		if offDays, err = ValidateWeeklyOffDays(in.WeeklyOffDays); err != nil {
			return nil, err
		}
	}

	var out *entity.OrganizationSettings
	err = uc.transactor.WithinTransaction(ctx, func(tx context.Context) error {
		if err := uc.repo.EnsureDefaults(tx, tenantID, actorID); err != nil {
			return err
		}
		// The row lock serialises this edit with concurrent code allocation, so the
		// issued-number check below cannot race a creation.
		cur, err := uc.repo.LockByTenantID(tx, tenantID)
		if err != nil {
			return err
		}
		next := cur.EmployeeCodeNextNumber
		if in.EmployeeCodeStartNumber != cur.EmployeeCodeStartNumber {
			// The start number can never be set at or below a number already issued.
			if cur.CodesIssued() && in.EmployeeCodeStartNumber <= cur.EmployeeCodeLastIssuedNumber {
				return errStartBelowIssued(cur.EmployeeCodeLastIssuedNumber + 1)
			}
			// The counter never moves backwards over issued numbers, and follows the
			// start number when it is raised (or nothing has been issued yet).
			if !cur.CodesIssued() || in.EmployeeCodeStartNumber > next {
				next = in.EmployeeCodeStartNumber
			}
		}
		updated := *cur
		updated.EmployeeCodePrefix = prefix
		updated.EmployeeCodePadding = in.EmployeeCodePadding
		updated.EmployeeCodeStartNumber = in.EmployeeCodeStartNumber
		updated.EmployeeCodeNextNumber = next
		if offDays != nil {
			updated.WeeklyOffDays = offDays
		}
		if setWelcome {
			updated.WelcomeMessage = welcome
		}
		if err := uc.repo.Update(tx, tenantID, actorID, &updated); err != nil {
			return err
		}
		out = &updated
		if uc.audit == nil {
			return nil
		}
		return uc.audit.Record(tx, tenantID, actorID, domainaudit.ActionOrganizationSettingsUpdate, domainaudit.EntityOrganizationSettings, updated.ID, map[string]any{
			"employee_code_prefix":       updated.EmployeeCodePrefix,
			"employee_code_padding":      updated.EmployeeCodePadding,
			"employee_code_start_number": updated.EmployeeCodeStartNumber,
			"welcome_message_set":        updated.WelcomeMessage != nil,
			"weekly_off_days":            updated.WeeklyOffDays,
			"previous": map[string]any{
				"employee_code_prefix":       cur.EmployeeCodePrefix,
				"employee_code_padding":      cur.EmployeeCodePadding,
				"employee_code_start_number": cur.EmployeeCodeStartNumber,
				"weekly_off_days":            cur.WeeklyOffDays,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// normalizeWelcomeMessage trims the message and validates its length. set is
// false when the field was omitted (leave unchanged); a blank message yields
// (nil, true) which clears it.
func normalizeWelcomeMessage(in *string) (msg *string, set bool, err error) {
	if in == nil {
		return nil, false, nil
	}
	m := strings.TrimSpace(*in)
	if m == "" {
		return nil, true, nil
	}
	if utf8.RuneCountInString(m) > entity.MaxWelcomeMessageLen {
		return nil, false, ErrWelcomeMessageTooLong
	}
	return &m, true, nil
}
