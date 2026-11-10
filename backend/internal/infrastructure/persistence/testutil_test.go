//go:build integration

package persistence

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/testutil/pgtest"
)

// setupTestDB returns a connection to the dedicated, migrated integration test
// database (see pgtest), never the dev database. It skips the test (fails in
// CI) if PostgreSQL is unreachable.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// .env lives at the repo root, 4 directories above this package, deeper
	// than config.Load's own built-in candidates reach.
	return pgtest.Open(t, "../../../..")
}

// createTestTenant inserts a tenant row directly (tenants carries no
// tenant_id itself) and registers cleanup to remove it (cascades to any
// users/tokens created under it, per the migrations' ON DELETE CASCADE FKs).
func createTestTenant(t *testing.T, db *gorm.DB, name string) *entity.Tenant {
	t.Helper()

	tenant := &entity.Tenant{
		ID:        uuid.New(),
		Name:      name,
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}
	t.Cleanup(func() {
		db.Unscoped().Where("id = ?", tenant.ID).Delete(&entity.Tenant{})
	})
	return tenant
}
