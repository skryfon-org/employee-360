package holiday

import (
	"strings"
	"time"
	"unicode/utf8"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

const (
	minHolidayYear = 2000
	maxHolidayYear = 2100
)

// NormalizeHolidayDate validates and normalizes a holiday date to UTC midnight.
// It returns domainerrors.ErrHolidayDateInvalid if the date is zero,
// or domainerrors.ErrHolidayYearInvalid if the year is outside [2000, 2100].
func NormalizeHolidayDate(d time.Time) (time.Time, error) {
	if d.IsZero() {
		return time.Time{}, domainerrors.ErrHolidayDateInvalid
	}
	norm := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	if norm.Year() < minHolidayYear || norm.Year() > maxHolidayYear {
		return time.Time{}, domainerrors.ErrHolidayYearInvalid
	}
	return norm, nil
}

func validateName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", domainerrors.ErrHolidayNameRequired
	}
	if utf8.RuneCountInString(trimmed) > 150 {
		return "", domainerrors.ErrHolidayNameTooLong
	}
	return trimmed, nil
}

func validateYear(year int) error {
	if year < minHolidayYear || year > maxHolidayYear {
		return domainerrors.ErrHolidayYearInvalid
	}
	return nil
}

func validateMonth(month *int) error {
	if month != nil && (*month < 1 || *month > 12) {
		return domainerrors.ErrHolidayMonthInvalid
	}
	return nil
}
