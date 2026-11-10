//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

func createTestUser(t *testing.T, db *gorm.DB, tenantID uuid.UUID, email string) *entity.User {
	t.Helper()
	now := time.Now().UTC()
	hash := "hashed-password"
	user := &entity.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        email,
		PasswordHash: &hash,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func TestGormUserRepository_CRUDRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormUserRepository(db)
	c := context.Background()

	tenant := createTestTenant(t, db, "crud-tenant-"+uuid.NewString())

	hash := "hashed-password"
	user := &entity.User{
		ID:           uuid.New(),
		TenantID:     tenant.ID,
		FirstName:    "Ada",
		LastName:     "Lovelace",
		Email:        "ada+" + uuid.NewString() + "@example.com",
		PasswordHash: &hash,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := repo.Create(c, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if user.TenantID != tenant.ID {
		t.Fatalf("expected TenantID to remain %s as set by the caller, got %s", tenant.ID, user.TenantID)
	}

	got, err := repo.GetByID(c, tenant.ID, user.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Email != user.Email {
		t.Errorf("expected email %q, got %q", user.Email, got.Email)
	}

	gotByEmail, err := repo.GetByTenantAndEmail(c, tenant.ID, user.Email)
	if err != nil {
		t.Fatalf("GetByTenantAndEmail failed: %v", err)
	}
	if gotByEmail.ID != user.ID {
		t.Errorf("expected id %s, got %s", user.ID, gotByEmail.ID)
	}

	got.FirstName = "Augusta"
	got.PasswordHash = nil // exercise zero-value field update
	if err := repo.Update(c, tenant.ID, got); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updated, err := repo.GetByID(c, tenant.ID, user.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if updated.FirstName != "Augusta" {
		t.Errorf("expected updated FirstName 'Augusta', got %q", updated.FirstName)
	}
	if updated.PasswordHash != nil {
		t.Errorf("expected PasswordHash to be cleared to nil, got %v", *updated.PasswordHash)
	}

	list, total, err := repo.List(c, tenant.ID, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total < 1 {
		t.Errorf("expected at least 1 user in List total, got %d", total)
	}
	found := false
	for _, u := range list {
		if u.ID == user.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("expected List to include created user %s", user.ID)
	}

	if err := repo.Delete(c, tenant.ID, user.ID, uuid.New()); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := repo.GetByID(c, tenant.ID, user.ID); err == nil {
		t.Error("expected GetByID to fail after Delete, got nil error")
	} else if err != domainerrors.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound after Delete, got: %v", err)
	}
}

func TestGormUserRepository_GetByIDWithRoles(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormUserRepository(db)
	c := context.Background()

	tenant := createTestTenant(t, db, "roles-tenant-"+uuid.NewString())

	user := createTestUser(t, db, tenant.ID, "role-user+"+uuid.NewString()+"@example.com")

	role := &entity.Role{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		Name:      entity.RoleAdmin,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	// Insert via raw SQL rather than db.Create(&entity.UserRole{...}): this
	// repository ticket only covers UserRepository/RefreshTokenRepository/
	// PasswordResetRepository, not UserRoleRepository, so there is no GORM
	// adapter for user_roles to reuse here.
	now := time.Now().UTC()
	if err := db.Exec(
		"INSERT INTO user_roles (id, tenant_id, user_id, role_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		uuid.New(), tenant.ID, user.ID, role.ID, now, now,
	).Error; err != nil {
		t.Fatalf("failed to create user_role: %v", err)
	}

	got, err := repo.GetByIDWithRoles(c, tenant.ID, user.ID)
	if err != nil {
		t.Fatalf("GetByIDWithRoles failed: %v", err)
	}
	if len(got.Roles) != 1 || got.Roles[0].Name != entity.RoleAdmin {
		t.Fatalf("expected exactly one 'admin' role, got %+v", got.Roles)
	}

	gotByEmail, err := repo.GetByTenantAndEmailWithRoles(c, tenant.ID, user.Email)
	if err != nil {
		t.Fatalf("GetByTenantAndEmailWithRoles failed: %v", err)
	}
	if len(gotByEmail.Roles) != 1 || gotByEmail.Roles[0].Name != entity.RoleAdmin {
		t.Fatalf("expected exactly one 'admin' role via email lookup, got %+v", gotByEmail.Roles)
	}
}

// TestGormUserRepository_TenantIsolation is the most important test in this
// file: it proves the fix from an earlier review (AC-4, colliding
// emails across tenants) actually holds end-to-end against a real database,
// not just against the usecase layer's in-memory mocks. The repository takes
// no ctx-based tenant resolution at all now -- every assertion below is
// expressed purely via the explicit tenantID parameter passed to each call.
func TestGormUserRepository_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormUserRepository(db)
	c := context.Background()

	tenantA := createTestTenant(t, db, "tenant-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "tenant-b-"+uuid.NewString())

	sharedEmail := "shared+" + uuid.NewString() + "@example.com"
	userA := createTestUser(t, db, tenantA.ID, sharedEmail)
	userB := createTestUser(t, db, tenantB.ID, sharedEmail)

	t.Run("GetByTenantAndEmail returns only the matching tenant's row", func(t *testing.T) {
		gotA, err := repo.GetByTenantAndEmail(c, tenantA.ID, sharedEmail)
		if err != nil {
			t.Fatalf("GetByTenantAndEmail(tenantA) failed: %v", err)
		}
		if gotA.ID != userA.ID {
			t.Errorf("expected tenant A's user %s, got %s", userA.ID, gotA.ID)
		}
		if gotA.ID == userB.ID {
			t.Fatal("tenant A's lookup returned tenant B's user")
		}

		gotB, err := repo.GetByTenantAndEmail(c, tenantB.ID, sharedEmail)
		if err != nil {
			t.Fatalf("GetByTenantAndEmail(tenantB) failed: %v", err)
		}
		if gotB.ID != userB.ID {
			t.Errorf("expected tenant B's user %s, got %s", userB.ID, gotB.ID)
		}
		if gotB.ID == userA.ID {
			t.Fatal("tenant B's lookup returned tenant A's user")
		}
	})

	t.Run("GetByID scoped to the wrong tenant cannot read another tenant's row", func(t *testing.T) {
		// tenantB passed explicitly, but the id belongs to tenant A's user.
		if _, err := repo.GetByID(c, tenantB.ID, userA.ID); err != domainerrors.ErrUserNotFound {
			t.Fatalf("expected ErrUserNotFound reading tenant A's user via tenant B's id, got: %v", err)
		}

		// Sanity check: the correct tenant id can read it.
		if _, err := repo.GetByID(c, tenantA.ID, userA.ID); err != nil {
			t.Fatalf("expected tenant A's own id to read its own user, got: %v", err)
		}
	})

	t.Run("Update scoped to the wrong tenant cannot mutate another tenant's row", func(t *testing.T) {
		mutated := *userA
		mutated.FirstName = "Hijacked"

		err := repo.Update(c, tenantB.ID, &mutated)
		if err != domainerrors.ErrUserNotFound {
			t.Fatalf("expected ErrUserNotFound updating tenant A's user via tenant B's id, got: %v", err)
		}

		// Confirm the row was NOT mutated.
		stillA, err := repo.GetByID(c, tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("failed to re-fetch tenant A's user: %v", err)
		}
		if stillA.FirstName == "Hijacked" {
			t.Fatal("cross-tenant Update mutated tenant A's row")
		}
	})

	t.Run("Delete scoped to the wrong tenant cannot remove another tenant's row", func(t *testing.T) {
		err := repo.Delete(c, tenantB.ID, userA.ID, uuid.New())
		if err != domainerrors.ErrUserNotFound {
			t.Fatalf("expected ErrUserNotFound deleting tenant A's user via tenant B's id, got: %v", err)
		}

		// Confirm the row still exists.
		if _, err := repo.GetByID(c, tenantA.ID, userA.ID); err != nil {
			t.Fatalf("expected tenant A's user to still exist after cross-tenant delete attempt, got: %v", err)
		}
	})
}
