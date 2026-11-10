// Package employee holds API DTOs and usecase inputs for employee management
// and the employee's own profile.
package employee

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// Optional distinguishes an absent JSON field (leave unchanged) from an explicit
// null (clear the value) from a provided value.
type Optional[T any] struct {
	Set   bool
	Null  bool
	Value T
}

// UnmarshalJSON implements json.Unmarshaler. It is only invoked when the key is
// present in the payload, so Set is always true here.
func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		o.Null = true
		var zero T
		o.Value = zero
		return nil
	}
	return json.Unmarshal(b, &o.Value)
}

// DateLayout is the wire format of date-only fields.
const DateLayout = "2006-01-02"

// Date is a calendar date (UTC midnight) serialised as YYYY-MM-DD.
type Date struct{ time.Time }

// UnmarshalJSON parses a strict YYYY-MM-DD string.
func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("date must be a YYYY-MM-DD string")
	}
	t, err := time.ParseInLocation(DateLayout, s, time.UTC)
	if err != nil {
		return fmt.Errorf("date must be formatted YYYY-MM-DD")
	}
	d.Time = t
	return nil
}

// MarshalJSON renders YYYY-MM-DD.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time.UTC().Format(DateLayout))
}

// FormatDate renders an optional time as a YYYY-MM-DD string pointer.
func FormatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(DateLayout)
	return &s
}
