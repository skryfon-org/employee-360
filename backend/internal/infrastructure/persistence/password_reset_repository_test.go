//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

func TestGormPasswordResetRepository_CRUDRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormPasswordResetRepository(db)

	tenant := createTestTenant(t, db, "pr-tenant-"+uuid.NewString())
	c := context.Background()
	user := createTestUser(t, db, tenant.ID, "pr-user+"+uuid.NewString()+"@example.com")

	now := time.Now().UTC()
	token := &entity.PasswordResetToken{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		UserID:    user.ID,
		TokenHash: "hash-" + uuid.NewString(),
		ExpiresAt: now.Add(15 * time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := repo.Create(c, token); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByTokenHash(c, token.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash failed: %v", err)
	}
	if got.ID != token.ID {
		t.Errorf("expected id %s, got %s", token.ID, got.ID)
	}
	if got.UsedAt != nil {
		t.Errorf("expected new token to be unused, got %v", got.UsedAt)
	}
	if !got.IsValid() {
		t.Error("expected freshly created token to be valid")
	}

	if err := repo.MarkAsUsed(c, token.ID); err != nil {
		t.Fatalf("MarkAsUsed failed: %v", err)
	}
	if err := repo.MarkAsUsed(c, token.ID); err != domainerrors.ErrTokenAlreadyUsed {
		t.Fatalf("second MarkAsUsed: expected ErrTokenAlreadyUsed, got %v", err)
	}
	used, err := repo.GetByTokenHash(c, token.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash after MarkAsUsed failed: %v", err)
	}
	if used.UsedAt == nil {
		t.Error("expected token to be marked as used")
	}
	if used.IsValid() {
		t.Error("expected used token to be invalid")
	}

	// A second, still-unused token for the same user, to exercise
	// InvalidateAllForUser independently of MarkAsUsed.
	token2 := &entity.PasswordResetToken{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		UserID:    user.ID,
		TokenHash: "hash-" + uuid.NewString(),
		ExpiresAt: now.Add(15 * time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(c, token2); err != nil {
		t.Fatalf("Create token2 failed: %v", err)
	}
	if err := repo.InvalidateAllForUser(c, user.ID); err != nil {
		t.Fatalf("InvalidateAllForUser failed: %v", err)
	}
	invalidated, err := repo.GetByTokenHash(c, token2.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash for token2 failed: %v", err)
	}
	if invalidated.UsedAt == nil {
		t.Error("expected token2 to be invalidated via InvalidateAllForUser")
	}

	// Expired token cleanup.
	expired := &entity.PasswordResetToken{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		UserID:    user.ID,
		TokenHash: "hash-" + uuid.NewString(),
		ExpiresAt: now.Add(-15 * time.Minute),
		CreatedAt: now.Add(-1 * time.Hour),
		UpdatedAt: now.Add(-1 * time.Hour),
	}
	if err := repo.Create(c, expired); err != nil {
		t.Fatalf("Create expired token failed: %v", err)
	}
	if err := repo.DeleteExpiredTokens(c, now); err != nil {
		t.Fatalf("DeleteExpiredTokens failed: %v", err)
	}
	if _, err := repo.GetByTokenHash(c, expired.TokenHash); err != domainerrors.ErrNotFound {
		t.Fatalf("expected expired token to be deleted, got err: %v", err)
	}
}

func TestGormPasswordResetRepository_GetByTokenHash_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormPasswordResetRepository(db)

	_, err := repo.GetByTokenHash(context.Background(), "does-not-exist-"+uuid.NewString())
	if err != domainerrors.ErrNotFound {
		t.Fatalf("expected ErrNotFound for unknown hash, got: %v", err)
	}
}
