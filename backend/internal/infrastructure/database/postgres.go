package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/skryfon-org/employee-360/backend/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DefaultConnTimeout is the default bounded timeout for verifying PostgreSQL connectivity.
const DefaultConnTimeout = 5 * time.Second

// Connect initializes a GORM PostgreSQL connection pool and verifies database reachability
// within a bounded timeout without retrying. Returns an error if the database is unreachable.
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := cfg.DSN()

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sql.DB from gorm: %w", err)
	}

	configureConnectionPool(sqlDB, cfg)

	// Bounded connectivity check (fail fast, no retry loop)
	timeout := cfg.ConnTimeout
	if timeout <= 0 {
		timeout = DefaultConnTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	return db, nil
}

// MustConnect establishes a GORM PostgreSQL connection or fails fast by logging
// "database unreachable: <error>" to stderr and exiting with exit code 1.
func MustConnect(cfg config.DatabaseConfig) *gorm.DB {
	db, err := Connect(cfg)
	if err != nil {
		FailFast(err)
	}
	return db
}

// FailFast writes a single log line containing "database unreachable: <err>" to stderr
// and exits the process immediately with non-zero exit code 1.
func FailFast(err error) {
	fmt.Fprintf(os.Stderr, "database unreachable: %v\n", err)
	os.Exit(1)
}

func configureConnectionPool(sqlDB *sql.DB, cfg config.DatabaseConfig) {
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
}
