// Package projectcommon holds validation and normalisation helpers shared by the
// project and allocation usecase packages. It must not import either of them.
package projectcommon

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }
func (e *validationError) Unwrap() error { return domainerrors.ErrInvalidProjectData }

// Invalid returns a client-correctable validation error; it matches
// domainerrors.ErrInvalidProjectData via errors.Is and its message is safe to show.
func Invalid(msg string) error { return &validationError{msg: msg} }

// Required returns the "field: is required" validation error.
func Required(field string) error { return Invalid(field + ": is required") }

// TooLong returns the "field: must not exceed N characters" validation error.
func TooLong(field string, max int) error {
	return Invalid(fmt.Sprintf("%s: must not exceed %d characters", field, max))
}

// ErrStartDateRequired is returned when a start date is missing.
var ErrStartDateRequired = Required("start_date")

// DateOnly normalises a time to its UTC calendar date.
func DateOnly(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// NormalizeOptional trims a nullable text field; blank becomes nil. A value
// longer than max runes yields tooLongErr.
func NormalizeOptional(raw *string, max int, tooLongErr error) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*raw)
	if v == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(v) > max {
		return nil, tooLongErr
	}
	return &v, nil
}
