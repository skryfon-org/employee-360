package database

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/gorm"
)

type txCtxKey struct{}

// ErrNoSQLTx is returned when a context does not carry an in-flight *sql.Tx.
var ErrNoSQLTx = errors.New("database: no in-flight sql transaction in context")

// WithTx returns a context carrying the given GORM transaction handle.
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// TxFromContext returns the in-flight GORM transaction carried by ctx, if any.
func TxFromContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txCtxKey{}).(*gorm.DB)
	return tx, ok && tx != nil
}

// DBFromContext returns the transaction carried by ctx when present, otherwise
// base, bound to ctx. Repositories use it so that they join the caller's
// transaction (started by GormTransactor) transparently.
func DBFromContext(ctx context.Context, base *gorm.DB) *gorm.DB {
	if tx, ok := TxFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return base.WithContext(ctx)
}

// SQLTxFromContext extracts the underlying *sql.Tx of the in-flight GORM
// transaction. It returns ErrNoSQLTx when ctx is not inside a transaction.
func SQLTxFromContext(ctx context.Context) (*sql.Tx, error) {
	tx, ok := TxFromContext(ctx)
	if !ok {
		return nil, ErrNoSQLTx
	}
	sqlTx, ok := tx.Statement.ConnPool.(*sql.Tx)
	if !ok {
		return nil, ErrNoSQLTx
	}
	return sqlTx, nil
}
