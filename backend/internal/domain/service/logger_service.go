package service

import "context"

// Logger defines a minimal, framework-agnostic logging port for usecases that
// must record an operational failure without bubbling it up as an error to
// their caller (e.g. enumeration-safe flows that always return nil regardless
// of internal failures). Infrastructure adapts this to the concrete logging
// library (zerolog, etc.) so the usecase layer stays free of framework
// dependencies.
type Logger interface {
	// Error records an operational failure. Implementations must not panic
	// or return an error themselves.
	Error(ctx context.Context, msg string, err error)
}
