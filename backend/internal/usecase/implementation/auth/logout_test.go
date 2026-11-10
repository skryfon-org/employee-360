package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	authtypes "github.com/skryfon-org/employee-360/backend/internal/types/auth"
)

func TestLogoutUseCase_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	tokenID := uuid.New()
	rawRefreshToken := "active-refresh-token"

	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{
		validateRefreshClaims: &domainservice.RefreshTokenClaims{
			UserID:   userID,
			TenantID: tenantID,
			TokenID:  tokenID,
		},
	}

	hashedToken := hashSvc.HashToken(rawRefreshToken)
	storedRT := &entity.RefreshToken{
		ID:        tokenID,
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: hashedToken,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = refreshTokenRepo.Create(context.Background(), storedRT)

	uc := NewLogoutUseCase(tokenSvc, hashSvc, refreshTokenRepo)

	err := uc.Execute(context.Background(), tenantID, userID, authtypes.LogoutRequest{
		RefreshToken: rawRefreshToken,
	})

	if err != nil {
		t.Fatalf("expected successful logout, got: %v", err)
	}

	if len(refreshTokenRepo.revokedTokenIDs) == 0 || refreshTokenRepo.revokedTokenIDs[0] != tokenID {
		t.Errorf("expected token %s to be revoked", tokenID)
	}
	if storedRT.RevokedAt == nil {
		t.Errorf("expected stored token RevokedAt to be set")
	}
}

func TestLogoutUseCase_InvalidOrMissingToken(t *testing.T) {
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{
		validateErr: domainerrors.ErrInvalidToken,
	}

	uc := NewLogoutUseCase(tokenSvc, hashSvc, refreshTokenRepo)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid signature", "corrupted.jwt.token"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := uc.Execute(context.Background(), uuid.New(), uuid.New(), authtypes.LogoutRequest{
				RefreshToken: tc.token,
			})
			if err != domainerrors.ErrInvalidToken {
				t.Errorf("expected ErrInvalidToken, got %v", err)
			}
		})
	}
}

func TestLogoutUseCase_IdempotentOnNonexistentToken(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	tokenSvc := &mockTokenService{
		validateRefreshClaims: &domainservice.RefreshTokenClaims{
			UserID:   userID,
			TenantID: tenantID,
			TokenID:  uuid.New(),
		},
	}
	hashSvc := &mockHashService{}
	refreshTokenRepo := newMockRefreshTokenRepository()

	uc := NewLogoutUseCase(tokenSvc, hashSvc, refreshTokenRepo)

	err := uc.Execute(context.Background(), tenantID, userID, authtypes.LogoutRequest{
		RefreshToken: "valid-jwt-but-not-in-db",
	})

	if err != nil {
		t.Fatalf("expected nil error on nonexistent token (idempotent logout), got %v", err)
	}
}

// TestLogoutUseCase_TokenOwnedByAnotherCaller proves a refresh token that does
// not belong to the authenticated caller's user/tenant is rejected with
// ErrInvalidToken and is NOT revoked.
func TestLogoutUseCase_TokenOwnedByAnotherCaller(t *testing.T) {
	tenantID := uuid.New()
	ownerID := uuid.New()
	tokenID := uuid.New()
	raw := "owners-refresh-token"

	hashSvc := &mockHashService{}
	refreshTokenRepo := newMockRefreshTokenRepository()
	stored := &entity.RefreshToken{
		ID: tokenID, TenantID: tenantID, UserID: ownerID,
		TokenHash: hashSvc.HashToken(raw), ExpiresAt: time.Now().Add(time.Hour),
	}
	_ = refreshTokenRepo.Create(context.Background(), stored)

	tokenSvc := &mockTokenService{
		validateRefreshClaims: &domainservice.RefreshTokenClaims{UserID: ownerID, TenantID: tenantID, TokenID: tokenID},
	}
	uc := NewLogoutUseCase(tokenSvc, hashSvc, refreshTokenRepo)

	tests := []struct {
		name     string
		tenantID uuid.UUID
		userID   uuid.UUID
	}{
		{"different user, same tenant", tenantID, uuid.New()},
		{"same user id, different tenant", uuid.New(), ownerID},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := uc.Execute(context.Background(), tc.tenantID, tc.userID, authtypes.LogoutRequest{RefreshToken: raw})
			if err != domainerrors.ErrInvalidToken {
				t.Errorf("expected ErrInvalidToken, got %v", err)
			}
			if stored.RevokedAt != nil || len(refreshTokenRepo.revokedTokenIDs) != 0 {
				t.Errorf("token must not be revoked by a non-owner")
			}
		})
	}
}
