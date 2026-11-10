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

func TestTokenRefreshUseCase_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	family := uuid.New()
	tokenID := uuid.New()
	rawRefreshToken := "valid-refresh-jwt-token"

	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{
		validateRefreshClaims: &domainservice.RefreshTokenClaims{
			UserID:   userID,
			TenantID: tenantID,
			TokenID:  tokenID,
			Family:   family,
			Roles:    []string{"admin"},
		},
	}

	user := &entity.User{
		ID:       userID,
		TenantID: tenantID,
		Email:    "alice@example.com",
		IsActive: true,
		Roles: []entity.Role{
			{ID: uuid.New(), TenantID: tenantID, Name: "admin"},
		},
	}
	_ = userRepo.Create(context.Background(), user)

	hashedToken := hashSvc.HashToken(rawRefreshToken)
	storedRT := &entity.RefreshToken{
		ID:        tokenID,
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: hashedToken,
		Family:    family,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = refreshTokenRepo.Create(context.Background(), storedRT)

	uc := NewTokenRefreshUseCase(userRepo, tokenSvc, hashSvc, refreshTokenRepo, &mockLogger{})

	output, err := uc.Execute(context.Background(), authtypes.TokenRefreshRequest{
		RefreshToken: rawRefreshToken,
	}, "127.0.0.1", "TestBrowser/1.0")

	if err != nil {
		t.Fatalf("expected successful token refresh, got: %v", err)
	}

	if output.AccessToken == "" || output.RefreshToken == "" {
		t.Fatalf("expected non-empty access and refresh tokens")
	}

	// Verify old token was revoked
	if storedRT.RevokedAt == nil {
		t.Errorf("expected old token to be marked as revoked")
	}

	// Verify new token was created with same family
	newHashedToken := hashSvc.HashToken(output.RefreshToken)
	newStoredRT, ok := refreshTokenRepo.tokensByHash[newHashedToken]
	if !ok {
		t.Fatalf("expected new refresh token to be persisted")
	}
	if newStoredRT.Family != family {
		t.Errorf("expected family %s to be preserved, got %s", family, newStoredRT.Family)
	}
	if newStoredRT.ID == tokenID {
		t.Errorf("expected new token to have a different ID than old token")
	}
}

func TestTokenRefreshUseCase_TokenReuseDetected(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	family := uuid.New()
	tokenID := uuid.New()
	rawRefreshToken := "reused-refresh-token"

	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{
		validateRefreshClaims: &domainservice.RefreshTokenClaims{
			UserID:   userID,
			TenantID: tenantID,
			TokenID:  tokenID,
			Family:   family,
		},
	}

	hashedToken := hashSvc.HashToken(rawRefreshToken)
	revokedAt := time.Now().Add(-10 * time.Minute)
	storedRT := &entity.RefreshToken{
		ID:        tokenID,
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: hashedToken,
		Family:    family,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		RevokedAt: &revokedAt,
	}
	_ = refreshTokenRepo.Create(context.Background(), storedRT)

	uc := NewTokenRefreshUseCase(userRepo, tokenSvc, hashSvc, refreshTokenRepo, &mockLogger{})

	_, err := uc.Execute(context.Background(), authtypes.TokenRefreshRequest{
		RefreshToken: rawRefreshToken,
	}, "", "")

	if err != domainerrors.ErrTokenRevoked {
		t.Errorf("expected ErrTokenRevoked on reuse, got: %v", err)
	}

	// Verify entire family was revoked
	if len(refreshTokenRepo.revokedFamilies) == 0 || refreshTokenRepo.revokedFamilies[0] != family {
		t.Errorf("expected entire family %s to be revoked on token reuse", family)
	}
}

func TestTokenRefreshUseCase_ExpiredToken(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	family := uuid.New()
	tokenID := uuid.New()
	rawRefreshToken := "expired-refresh-token"

	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{
		validateRefreshClaims: &domainservice.RefreshTokenClaims{
			UserID:   userID,
			TenantID: tenantID,
			TokenID:  tokenID,
			Family:   family,
		},
	}

	hashedToken := hashSvc.HashToken(rawRefreshToken)
	storedRT := &entity.RefreshToken{
		ID:        tokenID,
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: hashedToken,
		Family:    family,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	_ = refreshTokenRepo.Create(context.Background(), storedRT)

	uc := NewTokenRefreshUseCase(userRepo, tokenSvc, hashSvc, refreshTokenRepo, &mockLogger{})

	_, err := uc.Execute(context.Background(), authtypes.TokenRefreshRequest{
		RefreshToken: rawRefreshToken,
	}, "", "")

	if err != domainerrors.ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got: %v", err)
	}
}

func TestTokenRefreshUseCase_InvalidToken(t *testing.T) {
	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{
		validateErr: domainerrors.ErrInvalidToken,
	}

	uc := NewTokenRefreshUseCase(userRepo, tokenSvc, hashSvc, refreshTokenRepo, &mockLogger{})

	_, err := uc.Execute(context.Background(), authtypes.TokenRefreshRequest{
		RefreshToken: "invalid-token",
	}, "", "")

	if err != domainerrors.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

// TestTokenRefreshUseCase_RevokeFamilyFailureIsLogged proves that when
// revoking the token family on reuse detection fails, Execute still returns
// ErrTokenRevoked to the caller (the security outcome is unaffected) but the
// underlying repository failure is recorded via the Logger port rather than
// being silently discarded.
func TestTokenRefreshUseCase_RevokeFamilyFailureIsLogged(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	family := uuid.New()
	tokenID := uuid.New()
	rawRefreshToken := "reused-refresh-token"

	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{
		validateRefreshClaims: &domainservice.RefreshTokenClaims{
			UserID:   userID,
			TenantID: tenantID,
			TokenID:  tokenID,
			Family:   family,
		},
	}
	logger := &mockLogger{}

	hashedToken := hashSvc.HashToken(rawRefreshToken)
	revokedAt := time.Now().Add(-10 * time.Minute)
	storedRT := &entity.RefreshToken{
		ID:        tokenID,
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: hashedToken,
		Family:    family,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		RevokedAt: &revokedAt,
	}
	_ = refreshTokenRepo.Create(context.Background(), storedRT)

	revokeErr := domainerrors.ErrNotFound // reused only as a sentinel error for this test
	refreshTokenRepo.revokeFamilyErr = revokeErr

	uc := NewTokenRefreshUseCase(userRepo, tokenSvc, hashSvc, refreshTokenRepo, logger)

	_, err := uc.Execute(context.Background(), authtypes.TokenRefreshRequest{
		RefreshToken: rawRefreshToken,
	}, "", "")

	if err != domainerrors.ErrTokenRevoked {
		t.Errorf("expected ErrTokenRevoked on reuse even when RevokeFamily fails, got: %v", err)
	}

	if len(logger.errorCalls) != 1 {
		t.Fatalf("expected the RevokeFamily failure to be logged exactly once, got %d calls", len(logger.errorCalls))
	}
	if logger.errorCalls[0].err != revokeErr {
		t.Errorf("expected logged error to be the repository's error, got %v", logger.errorCalls[0].err)
	}
}

// racingRefreshTokenRepo simulates losing the rotation race: GetByTokenHash
// still sees the token as live, but the atomic Revoke reports it was already
// revoked by a concurrent request.
type racingRefreshTokenRepo struct {
	*mockRefreshTokenRepository
}

func (r *racingRefreshTokenRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	return domainerrors.ErrTokenAlreadyRevoked
}

func TestTokenRefreshUseCase_LostRotationRaceTreatedAsReuse(t *testing.T) {
	tenantID, userID, family, tokenID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	raw := "raced-refresh-token"

	base := newMockRefreshTokenRepository()
	refreshTokenRepo := &racingRefreshTokenRepo{base}
	userRepo := newMockUserRepository()
	_ = userRepo.Create(context.Background(), &entity.User{ID: userID, TenantID: tenantID, Email: "a@example.com", IsActive: true})
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{
		validateRefreshClaims: &domainservice.RefreshTokenClaims{UserID: userID, TenantID: tenantID, TokenID: tokenID, Family: family},
	}
	_ = base.Create(context.Background(), &entity.RefreshToken{
		ID: tokenID, TenantID: tenantID, UserID: userID, TokenHash: hashSvc.HashToken(raw),
		Family: family, ExpiresAt: time.Now().Add(time.Hour),
	})

	uc := NewTokenRefreshUseCase(userRepo, tokenSvc, hashSvc, refreshTokenRepo, &mockLogger{})
	out, err := uc.Execute(context.Background(), authtypes.TokenRefreshRequest{RefreshToken: raw}, "", "")

	if err != domainerrors.ErrTokenRevoked {
		t.Fatalf("expected ErrTokenRevoked, got %v", err)
	}
	if out != nil {
		t.Errorf("expected no token pair on lost race")
	}
	if len(base.revokedFamilies) != 1 || base.revokedFamilies[0] != family {
		t.Errorf("expected family %s revoked, got %v", family, base.revokedFamilies)
	}
	if len(base.tokensByID) != 1 {
		t.Errorf("expected no child token to be created, repo has %d tokens", len(base.tokensByID))
	}
}

func TestMockRefreshTokenRepository_SecondRevokeReturnsSentinel(t *testing.T) {
	repo := newMockRefreshTokenRepository()
	id := uuid.New()
	_ = repo.Create(context.Background(), &entity.RefreshToken{ID: id, TokenHash: "h"})
	if err := repo.Revoke(context.Background(), id); err != nil {
		t.Fatalf("first revoke: %v", err)
	}
	if err := repo.Revoke(context.Background(), id); err != domainerrors.ErrTokenAlreadyRevoked {
		t.Fatalf("expected sentinel on second revoke, got %v", err)
	}
}
