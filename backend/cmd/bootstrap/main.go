// Command bootstrap seeds the initial system tenant, core roles, and Super Admin.
//
// It is idempotent and safe to run repeatedly. The Super Admin credentials come
// from configuration (BOOTSTRAP_SUPER_ADMIN_EMAIL / BOOTSTRAP_SUPER_ADMIN_PASSWORD)
// and are never hardcoded; the command fails if they are unset.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database/seeder"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	opts := seeder.Options{
		SystemTenantName:   cfg.Bootstrap.SystemTenantName,
		SuperAdminEmail:    cfg.Bootstrap.SuperAdminEmail,
		SuperAdminPassword: cfg.Bootstrap.SuperAdminPassword,
	}
	// Fail on missing credentials before touching the database.
	if err := opts.Validate(); err != nil {
		return err
	}

	db, err := database.Connect(cfg.Database)
	if err != nil {
		database.FailFast(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		defer func() { _ = sqlDB.Close() }()
	}

	res, err := seeder.New(db, service.NewHashService()).Run(context.Background(), opts)
	if err != nil {
		return err
	}

	fmt.Printf("bootstrap complete: tenant %s (created=%t), roles created=%v, super admin %s (created=%t), domain registered=%t\n",
		res.TenantID, res.TenantCreated, res.RolesCreated, res.AdminID, res.AdminCreated, res.DomainCreated)
	return nil
}
