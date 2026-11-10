//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

func TestGormRefreshTokenRepository_CRUDRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRefreshTokenRepository(db)

	tenant := createTestTenant(t, db, "rt-tenant-"+uuid.NewString())
	c := context.Background()
	user := createTestUser(t, db, tenant.ID, "rt-user+"+uuid.NewString()+"@example.com")

	now := time.Now().UTC()
	family := uuid.New()
	token := &entity.RefreshToken{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		UserID:    user.ID,
		TokenHash: "hash-" + uuid.NewString(),
		Family:    family,
		ExpiresAt: now.Add(24 * time.Hour),
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
	if got.RevokedAt != nil {
		t.Errorf("expected new token to not be revoked, got %v", got.RevokedAt)
	}

	if err := repo.Revoke(c, token.ID); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
	if err := repo.Revoke(c, token.ID); err != domainerrors.ErrTokenAlreadyRevoked {
		t.Fatalf("second Revoke: expected ErrTokenAlreadyRevoked, got %v", err)
	}
	revoked, err := repo.GetByTokenHash(c, token.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash after revoke failed: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Error("expected token to be revoked")
	}

	// Second token in the same family, to exercise RevokeFamily.
	token2 := &entity.RefreshToken{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		UserID:    user.ID,
		TokenHash: "hash-" + uuid.NewString(),
		Family:    family,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(c, token2); err != nil {
		t.Fatalf("Create token2 failed: %v", err)
	}
	if err := repo.RevokeFamily(c, family); err != nil {
		t.Fatalf("RevokeFamily failed: %v", err)
	}
	revoked2, err := repo.GetByTokenHash(c, token2.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash for token2 failed: %v", err)
	}
	if revoked2.RevokedAt == nil {
		t.Error("expected token2 to be revoked via RevokeFamily")
	}

	// Third token for the user, to exercise RevokeAllForUser.
	token3 := &entity.RefreshToken{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		UserID:    user.ID,
		TokenHash: "hash-" + uuid.NewString(),
		Family:    uuid.New(),
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(c, token3); err != nil {
		t.Fatalf("Create token3 failed: %v", err)
	}
	if err := repo.RevokeAllForUser(c, user.ID); err != nil {
		t.Fatalf("RevokeAllForUser failed: %v", err)
	}
	revoked3, err := repo.GetByTokenHash(c, token3.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash for token3 failed: %v", err)
	}
	if revoked3.RevokedAt == nil {
		t.Error("expected token3 to be revoked via RevokeAllForUser")
	}

	// Expired token cleanup.
	expired := &entity.RefreshToken{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		UserID:    user.ID,
		TokenHash: "hash-" + uuid.NewString(),
		Family:    uuid.New(),
		ExpiresAt: now.Add(-24 * time.Hour),
		CreatedAt: now.Add(-48 * time.Hour),
		UpdatedAt: now.Add(-48 * time.Hour),
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

func TestGormRefreshTokenRepository_GetByTokenHash_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRefreshTokenRepository(db)

	_, err := repo.GetByTokenHash(context.Background(), "does-not-exist-"+uuid.NewString())
	if err != domainerrors.ErrNotFound {
		t.Fatalf("expected ErrNotFound for unknown hash, got: %v", err)
	}
}
