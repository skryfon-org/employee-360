// Package leavecommon holds helpers shared by the leave type and leave request
// usecase packages. It must not import either of them.
package leavecommon

import (
	"strings"
	"time"
	"unicode/utf8"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

// Clock returns the current reference time.
type Clock func() time.Time

// RealClock returns the current UTC time.
func RealClock() time.Time { return time.Now().UTC() }

// DateOnly truncates a time to UTC midnight.
func DateOnly(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }
func (e *validationError) Is(target error) bool {
	return target == domainerrors.ErrInvalidLeaveData
}

// Invalid returns a client-correctable validation error; it matches
// domainerrors.ErrInvalidLeaveData via errors.Is and its message is safe to show.
func Invalid(msg string) error { return &validationError{msg: msg} }

// NormalizeText trims s and rejects values longer than max characters.
func NormalizeText(s, field string, max int) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > max {
		return "", Invalid(field + " is too long")
	}
	return s, nil
}

// AppURLs holds the public base URLs used to build links in leave emails.
// Admin and Employee are optional and fall back to Default when empty.
type AppURLs struct {
	Default  string
	Admin    string
	Employee string
}

func (a AppURLs) base(u string) string {
	if u == "" {
		u = a.Default
	}
	return strings.TrimRight(u, "/")
}

// ReviewLink is the page an approver reviews a request on: the admin app for
// admin approvers, the employee app (team requests) for managers.
func (a AppURLs) ReviewLink(approverKind string) string {
	if approverKind == "admin" {
		return a.base(a.Admin) + "/leave"
	}
	return a.base(a.Employee) + "/leave"
}

// EmployeeLeaveLink is the employee's own leave page.
func (a AppURLs) EmployeeLeaveLink() string { return a.base(a.Employee) + "/leave" }
