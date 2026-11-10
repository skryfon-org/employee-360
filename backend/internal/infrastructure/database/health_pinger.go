package database

import (
	"context"
	"fmt"

	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"gorm.io/gorm"
)

// GormDatabasePinger implements service.DatabasePinger on top of a GORM
// connection, so usecases can check database liveness without depending
// on GORM directly.
type GormDatabasePinger struct {
	db *gorm.DB
}

var _ service.DatabasePinger = (*GormDatabasePinger)(nil)

// NewGormDatabasePinger constructs a GormDatabasePinger wrapping db.
func NewGormDatabasePinger(db *gorm.DB) *GormDatabasePinger {
	return &GormDatabasePinger{db: db}
}

// Ping verifies database reachability by pinging the underlying sql.DB.
func (p *GormDatabasePinger) Ping(ctx context.Context) error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to retrieve sql.DB from gorm: %w", err)
	}
	return sqlDB.PingContext(ctx)
}
