//go:build integration

package persistence

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

func createTestTenantDomain(t *testing.T, db *gorm.DB, tenantID uuid.UUID, domain string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO tenant_domains (id, tenant_id, domain) VALUES (?, ?, ?)",
		uuid.New(), tenantID, domain,
	).Error; err != nil {
		t.Fatalf("failed to create tenant domain: %v", err)
	}
	// Rows are removed by the tenant's ON DELETE CASCADE cleanup.
}

func TestGormTenantDomainRepository_FindTenantByDomain(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormTenantDomainRepository(db)
	c := context.Background()

	suffix := uuid.NewString()
	tenantA := createTestTenant(t, db, "td-a-"+suffix)
	tenantB := createTestTenant(t, db, "td-b-"+suffix)
	domainA := "a-" + suffix + ".example.com"
	domainB := "b-" + suffix + ".example.com"
	createTestTenantDomain(t, db, tenantA.ID, domainA)
	createTestTenantDomain(t, db, tenantB.ID, domainB)

	t.Run("unknown domain returns ErrTenantNotFound", func(t *testing.T) {
		got, err := repo.FindTenantByDomain(c, "unknown-"+suffix+".example.com")
		if !errors.Is(err, domainerrors.ErrTenantNotFound) {
			t.Fatalf("expected ErrTenantNotFound, got %v", err)
		}
		if got != nil {
			t.Errorf("expected nil tenant, got %+v", got)
		}
	})

	t.Run("case-insensitive match", func(t *testing.T) {
		got, err := repo.FindTenantByDomain(c, "A-"+suffix+".Example.COM")
		if err != nil {
			t.Fatalf("FindTenantByDomain failed: %v", err)
		}
		if got.ID != tenantA.ID {
			t.Errorf("expected tenant %s, got %s", tenantA.ID, got.ID)
		}
	})

	t.Run("distinct domains resolve to their own tenants", func(t *testing.T) {
		gotA, err := repo.FindTenantByDomain(c, domainA)
		if err != nil {
			t.Fatalf("FindTenantByDomain(A) failed: %v", err)
		}
		gotB, err := repo.FindTenantByDomain(c, domainB)
		if err != nil {
			t.Fatalf("FindTenantByDomain(B) failed: %v", err)
		}
		if gotA.ID != tenantA.ID {
			t.Errorf("domain A: expected tenant %s, got %s", tenantA.ID, gotA.ID)
		}
		if gotB.ID != tenantB.ID {
			t.Errorf("domain B: expected tenant %s, got %s", tenantB.ID, gotB.ID)
		}
	})

	t.Run("inactive tenant is excluded", func(t *testing.T) {
		inactive := createTestTenant(t, db, "td-inactive-"+suffix)
		domain := "inactive-" + suffix + ".example.com"
		createTestTenantDomain(t, db, inactive.ID, domain)
		if err := db.Exec("UPDATE tenants SET is_active = FALSE WHERE id = ?", inactive.ID).Error; err != nil {
			t.Fatalf("failed to deactivate tenant: %v", err)
		}

		got, err := repo.FindTenantByDomain(c, domain)
		if !errors.Is(err, domainerrors.ErrTenantNotFound) {
			t.Fatalf("expected ErrTenantNotFound for inactive tenant, got tenant=%v err=%v", got, err)
		}
	})

	t.Run("soft-deleted tenant is excluded", func(t *testing.T) {
		tn := createTestTenant(t, db, "td-del-"+suffix)
		domain := "deleted-tenant-" + suffix + ".example.com"
		createTestTenantDomain(t, db, tn.ID, domain)
		require.NoError(t, db.Exec("UPDATE tenants SET deleted_at = NOW() WHERE id = ?", tn.ID).Error)

		got, err := repo.FindTenantByDomain(c, domain)
		if !errors.Is(err, domainerrors.ErrTenantNotFound) {
			t.Fatalf("expected ErrTenantNotFound for soft-deleted tenant, got tenant=%v err=%v", got, err)
		}
	})

	t.Run("soft-deleted domain row is excluded", func(t *testing.T) {
		tn := createTestTenant(t, db, "td-ddel-"+suffix)
		domain := "deleted-domain-" + suffix + ".example.com"
		createTestTenantDomain(t, db, tn.ID, domain)
		require.NoError(t, db.Exec("UPDATE tenant_domains SET deleted_at = NOW() WHERE domain = ?", domain).Error)

		got, err := repo.FindTenantByDomain(c, domain)
		if !errors.Is(err, domainerrors.ErrTenantNotFound) {
			t.Fatalf("expected ErrTenantNotFound for soft-deleted domain, got tenant=%v err=%v", got, err)
		}
	})
}

func TestGormTenantDomainRepository_DomainBelongsToTenant(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormTenantDomainRepository(db)
	c := context.Background()

	suffix := uuid.NewString()
	tenantA := createTestTenant(t, db, "tdb-a-"+suffix)
	tenantB := createTestTenant(t, db, "tdb-b-"+suffix)
	domainA := "a-" + suffix + ".example.com"
	createTestTenantDomain(t, db, tenantA.ID, domainA)

	ok, err := repo.DomainBelongsToTenant(c, tenantA.ID, domainA)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.DomainBelongsToTenant(c, tenantA.ID, strings.ToUpper(domainA))
	require.NoError(t, err)
	require.True(t, ok, "case-insensitive")
	ok, err = repo.DomainBelongsToTenant(c, tenantB.ID, domainA)
	require.NoError(t, err)
	require.False(t, ok, "other tenant's domain")

	require.NoError(t, db.Exec("UPDATE tenant_domains SET deleted_at = NOW() WHERE domain = ?", domainA).Error)
	ok, err = repo.DomainBelongsToTenant(c, tenantA.ID, domainA)
	require.NoError(t, err)
	require.False(t, ok, "soft-deleted domain")
}
