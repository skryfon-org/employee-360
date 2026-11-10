//go:build integration

package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

func TestGormTenantReader_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormTenantReader(db)
	c := context.Background()

	live := createTestTenant(t, db, "reader-live-"+uuid.NewString())
	deleted := createTestTenant(t, db, "reader-deleted-"+uuid.NewString())
	inactive := createTestTenant(t, db, "reader-inactive-"+uuid.NewString())

	if err := db.Exec("UPDATE tenants SET deleted_at = ? WHERE id = ?", time.Now().UTC(), deleted.ID).Error; err != nil {
		t.Fatalf("soft-delete tenant: %v", err)
	}
	if err := db.Exec("UPDATE tenants SET is_active = FALSE WHERE id = ?", inactive.ID).Error; err != nil {
		t.Fatalf("deactivate tenant: %v", err)
	}

	got, err := repo.GetByID(c, live.ID)
	if err != nil || got.ID != live.ID || !got.IsActive {
		t.Fatalf("live tenant: got %+v err %v", got, err)
	}

	if _, err := repo.GetByID(c, deleted.ID); !errors.Is(err, domainerrors.ErrTenantNotFound) {
		t.Errorf("soft-deleted tenant: err = %v, want ErrTenantNotFound", err)
	}
	if _, err := repo.GetByID(c, uuid.New()); !errors.Is(err, domainerrors.ErrTenantNotFound) {
		t.Errorf("unknown tenant: err = %v, want ErrTenantNotFound", err)
	}

	// Inactive tenants are returned (the usecase decides policy) with IsActive=false.
	got, err = repo.GetByID(c, inactive.ID)
	if err != nil || got.IsActive {
		t.Errorf("inactive tenant: got %+v err %v", got, err)
	}
}
