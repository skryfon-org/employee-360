package request

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	leavecommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave"
)

func validStatus(s string) bool {
	for _, v := range entity.LeaveStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// parseStatusFilter validates an optional status filter.
func parseStatusFilter(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s != "" && !validStatus(s) {
		return "", leavecommon.Invalid("invalid status filter")
	}
	return s, nil
}

func parseUUIDFilter(s, field string) (*uuid.UUID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, leavecommon.Invalid("invalid " + field)
	}
	return &id, nil
}

func parseDateFilter(s, field string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation(emptypes.DateLayout, s, time.UTC)
	if err != nil {
		return nil, leavecommon.Invalid(field + " must be formatted YYYY-MM-DD")
	}
	return &t, nil
}

// parseDateRange parses optional from/to filters; either may be empty, and to
// must not be before from.
func parseDateRange(fromS, toS string) (from, to *time.Time, err error) {
	if from, err = parseDateFilter(fromS, "from"); err != nil {
		return nil, nil, err
	}
	if to, err = parseDateFilter(toS, "to"); err != nil {
		return nil, nil, err
	}
	if from != nil && to != nil && to.Before(*from) {
		return nil, nil, leavecommon.Invalid("to must not be before from")
	}
	return from, to, nil
}

// startGraceDays is the number of days before the server's UTC today that a
// start date is still accepted. Clients pick dates from their LOCAL calendar
// day, which can be one day behind UTC (e.g. the Americas in the evening), so a
// strict UTC check would reject a date that looks valid to the user.
const startGraceDays = 1

// ValidateStartWindow enforces start >= (UTC today - startGraceDays) using the
// usecase clock; there is no upper bound.
func ValidateStartWindow(start, now time.Time) error {
	if start.Before(leavecommon.DateOnly(now).AddDate(0, 0, -startGraceDays)) {
		return domainerrors.ErrLeaveStartTooOld
	}
	return nil
}

// ValidateDayPart checks the day part against the date span. Half-day parts
// are only valid for a single day.
func ValidateDayPart(dayPart string, start, end time.Time) (string, error) {
	if dayPart == "" {
		dayPart = entity.LeaveDayPartFull
	}
	ok := false
	for _, v := range entity.LeaveDayParts {
		if v == dayPart {
			ok = true
		}
	}
	if !ok {
		return "", leavecommon.Invalid("day_part must be one of full, first_half, second_half")
	}
	if dayPart != entity.LeaveDayPartFull && !start.Equal(end) {
		return "", leavecommon.Invalid("half-day leave requires start_date and end_date to be the same day")
	}
	return dayPart, nil
}

// resolveApprovers returns the caller's active manager (employee_profiles.manager_id)
// followed by the active tenant admins, excluding the caller. A manager who is
// also an admin appears once, as the manager.
func resolveApprovers(ctx context.Context, profiles repository.EmployeeProfileRepository, approvers repository.LeaveApproverRepository, tenantID, userID uuid.UUID) ([]*entity.LeaveApprover, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	out := []*entity.LeaveApprover{}
	seen := map[uuid.UUID]bool{userID: true}

	prof, err := profiles.GetByUserID(ctx, tenantID, userID)
	if err != nil && !errors.Is(err, domainerrors.ErrEmployeeProfileNotFound) {
		return nil, err
	}
	if err == nil && prof.ManagerID != nil && *prof.ManagerID != userID {
		m, ok, err := approvers.GetActiveUser(ctx, tenantID, *prof.ManagerID)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, m)
			seen[m.UserID] = true
		}
	}

	admins, err := approvers.ListActiveAdmins(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, a := range admins {
		if !seen[a.UserID] {
			seen[a.UserID] = true
			out = append(out, a)
		}
	}
	return out, nil
}
