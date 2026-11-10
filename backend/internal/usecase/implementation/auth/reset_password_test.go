package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	authtypes "github.com/skryfon-org/employee-360/backend/internal/types/auth"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
)

func TestResetPasswordUseCase_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	tokenID := uuid.New()
	plainToken := "secure-plain-reset-token-12345"

	userRepo := newMockUserRepository()
	resetRepo := newMockPasswordResetRepository()
	refreshRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	transactor := ucshared.NewNopTransactor()

	user := &entity.User{
		ID:       userID,
		TenantID: tenantID,
		Email:    "alice@example.com",
		IsActive: true,
	}
	_ = userRepo.Create(context.Background(), user)

	tokenHash := hashSvc.HashToken(plainToken)
	resetToken := &entity.PasswordResetToken{
		ID:        tokenID,
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = resetRepo.Create(context.Background(), resetToken)

	// Create some active refresh tokens for this user across different sessions
	rt1 := &entity.RefreshToken{ID: uuid.New(), TenantID: tenantID, UserID: userID, TokenHash: "h1", Family: uuid.New(), ExpiresAt: time.Now().Add(24 * time.Hour)}
	rt2 := &entity.RefreshToken{ID: uuid.New(), TenantID: tenantID, UserID: userID, TokenHash: "h2", Family: uuid.New(), ExpiresAt: time.Now().Add(24 * time.Hour)}
	_ = refreshRepo.Create(context.Background(), rt1)
	_ = refreshRepo.Create(context.Background(), rt2)

	uc := NewResetPasswordUseCase(userRepo, resetRepo, refreshRepo, hashSvc, transactor)

	err := uc.Execute(context.Background(), authtypes.ResetPasswordRequest{
		Token:       plainToken,
		NewPassword: "BrandNewSecurePassword123!",
	})

	if err != nil {
		t.Fatalf("expected successful reset password, got: %v", err)
	}

	// Verify token was marked as used
	if len(resetRepo.usedTokenIDs) == 0 || resetRepo.usedTokenIDs[0] != tokenID {
		t.Errorf("expected reset token %s to be marked as used", tokenID)
	}
	if resetToken.UsedAt == nil {
		t.Errorf("expected reset token UsedAt to be set")
	}

	// Verify user's password was updated with hash
	updatedUser, _ := userRepo.GetByID(context.Background(), tenantID, userID)
	if updatedUser.PasswordHash == nil || *updatedUser.PasswordHash != "hashed_BrandNewSecurePassword123!" {
		t.Errorf("expected updated password hash, got %v", updatedUser.PasswordHash)
	}

	// Acceptance criterion: Successful password reset revokes every existing refresh token for that user, not just the current session's
	if len(refreshRepo.revokedUserIDs) == 0 || refreshRepo.revokedUserIDs[0] != userID {
		t.Errorf("expected all refresh tokens for user %s to be revoked", userID)
	}
	if rt1.RevokedAt == nil || rt2.RevokedAt == nil {
		t.Errorf("expected all user sessions to be invalidated")
	}
}

func TestResetPasswordUseCase_InvalidOrExpiredToken(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	userRepo := newMockUserRepository()
	resetRepo := newMockPasswordResetRepository()
	refreshRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	transactor := ucshared.NewNopTransactor()

	usedAt := time.Now().Add(-5 * time.Minute)
	usedToken := &entity.PasswordResetToken{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: hashSvc.HashToken("already-used-token"),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		UsedAt:    &usedAt,
	}
	_ = resetRepo.Create(context.Background(), usedToken)

	expiredToken := &entity.PasswordResetToken{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: hashSvc.HashToken("expired-token"),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	_ = resetRepo.Create(context.Background(), expiredToken)

	uc := NewResetPasswordUseCase(userRepo, resetRepo, refreshRepo, hashSvc, transactor)

	tests := []struct {
		name        string
		token       string
		newPassword string
		expectedErr error
	}{
		{"empty token", "", "ValidPassword123!", domainerrors.ErrInvalidToken},
		{"nonexistent token", "unknown-token", "ValidPassword123!", domainerrors.ErrInvalidToken},
		{"already used token", "already-used-token", "ValidPassword123!", domainerrors.ErrInvalidToken},
		{"expired token", "expired-token", "ValidPassword123!", domainerrors.ErrInvalidToken},
		{"password too short", "already-used-token", "short", domainerrors.ErrInvalidPassword},
		{"password over 72 bytes", "already-used-token", strings.Repeat("a", 73), domainerrors.ErrInvalidPassword},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := uc.Execute(context.Background(), authtypes.ResetPasswordRequest{
				Token:       tc.token,
				NewPassword: tc.newPassword,
			})
			if err != tc.expectedErr {
				t.Errorf("expected %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestResetPasswordUseCase_InactiveUser(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	plainToken := "valid-token-for-inactive-user"

	userRepo := newMockUserRepository()
	resetRepo := newMockPasswordResetRepository()
	refreshRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	transactor := ucshared.NewNopTransactor()

	user := &entity.User{
		ID:       userID,
		TenantID: tenantID,
		Email:    "inactive@example.com",
		IsActive: false,
	}
	_ = userRepo.Create(context.Background(), user)

	resetToken := &entity.PasswordResetToken{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: hashSvc.HashToken(plainToken),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	_ = resetRepo.Create(context.Background(), resetToken)

	uc := NewResetPasswordUseCase(userRepo, resetRepo, refreshRepo, hashSvc, transactor)

	err := uc.Execute(context.Background(), authtypes.ResetPasswordRequest{
		Token:       plainToken,
		NewPassword: "BrandNewSecurePassword123!",
	})

	if err != domainerrors.ErrUserInactive {
		t.Errorf("expected ErrUserInactive, got %v", err)
	}
}

func TestResetPasswordUseCase_DoubleUseFails(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	plain := "double-use-reset-token"

	userRepo := newMockUserRepository()
	resetRepo := newMockPasswordResetRepository()
	refreshRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	_ = userRepo.Create(context.Background(), &entity.User{ID: userID, TenantID: tenantID, Email: "a@example.com", IsActive: true})
	_ = resetRepo.Create(context.Background(), &entity.PasswordResetToken{
		ID: uuid.New(), TenantID: tenantID, UserID: userID, TokenHash: hashSvc.HashToken(plain),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	})
	uc := NewResetPasswordUseCase(userRepo, resetRepo, refreshRepo, hashSvc, ucshared.NewNopTransactor())
	req := authtypes.ResetPasswordRequest{Token: plain, NewPassword: "BrandNewSecurePassword123!"}

	if err := uc.Execute(context.Background(), req); err != nil {
		t.Fatalf("first use should succeed: %v", err)
	}
	if err := uc.Execute(context.Background(), req); err != domainerrors.ErrInvalidToken {
		t.Fatalf("second use should fail with ErrInvalidToken, got %v", err)
	}

	// Lost race: token read as valid but MarkAsUsed reports already used.
	racing := &racingResetRepo{newMockPasswordResetRepository()}
	_ = racing.Create(context.Background(), &entity.PasswordResetToken{
		ID: uuid.New(), TenantID: tenantID, UserID: userID, TokenHash: hashSvc.HashToken("race"),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	})
	uc2 := NewResetPasswordUseCase(userRepo, racing, refreshRepo, hashSvc, ucshared.NewNopTransactor())
	if err := uc2.Execute(context.Background(), authtypes.ResetPasswordRequest{Token: "race", NewPassword: "BrandNewSecurePassword123!"}); err != domainerrors.ErrInvalidToken {
		t.Fatalf("lost race should map to ErrInvalidToken, got %v", err)
	}
}

type racingResetRepo struct{ *mockPasswordResetRepository }

func (r *racingResetRepo) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	return domainerrors.ErrTokenAlreadyUsed
}
