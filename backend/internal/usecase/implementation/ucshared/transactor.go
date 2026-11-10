package ucshared

import "context"

// Transactor defines an interface for executing units of work within a database transaction.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// NopTransactor is a no-op Transactor implementation used for testing and in-memory execution.
type NopTransactor struct{}

// NewNopTransactor creates a new NopTransactor.
func NewNopTransactor() *NopTransactor {
	return &NopTransactor{}
}

// WithinTransaction runs fn directly with the provided context.
func (n *NopTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
