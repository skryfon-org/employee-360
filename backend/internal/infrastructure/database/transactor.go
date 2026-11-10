package database

import (
	"context"

	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	"gorm.io/gorm"
)

// GormTransactor implements ucshared.Transactor on top of GORM. The transaction
// is propagated through the context so repositories and the EventPublisher
// share it without leaking *gorm.DB / *sql.Tx into the usecase layer.
type GormTransactor struct {
	db *gorm.DB
}

var _ ucshared.Transactor = (*GormTransactor)(nil)

// NewGormTransactor constructs a GormTransactor.
func NewGormTransactor(db *gorm.DB) *GormTransactor {
	return &GormTransactor{db: db}
}

// WithinTransaction runs fn in a transaction: commit on nil, rollback on error
// or panic. If ctx already carries a transaction, fn joins it (the outermost
// call owns commit/rollback).
func (t *GormTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := TxFromContext(ctx); ok {
		return fn(ctx)
	}
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(WithTx(ctx, tx))
	})
}
