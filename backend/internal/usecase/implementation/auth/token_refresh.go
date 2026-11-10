package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	authtypes "github.com/skryfon-org/employee-360/backend/internal/types/auth"
	authusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/auth"
)

// TokenRefreshUseCaseImpl implements authusecase.TokenRefreshUseCase.
type TokenRefreshUseCaseImpl struct {
	userRepo         repository.UserRepository
	tokenService     service.TokenService
	hashService      service.HashService
	refreshTokenRepo repository.RefreshTokenRepository
	logger           service.Logger
	refreshExpiry    time.Duration
}

var _ authusecase.TokenRefreshUseCase = (*TokenRefreshUseCaseImpl)(nil)

// NewTokenRefreshUseCase constructs a new TokenRefreshUseCaseImpl.
func NewTokenRefreshUseCase(
	userRepo repository.UserRepository,
	tokenService service.TokenService,
	hashService service.HashService,
	refreshTokenRepo repository.RefreshTokenRepository,
	logger service.Logger,
) *TokenRefreshUseCaseImpl {
	return &TokenRefreshUseCaseImpl{
		userRepo:         userRepo,
		tokenService:     tokenService,
		hashService:      hashService,
		refreshTokenRepo: refreshTokenRepo,
		logger:           logger,
		refreshExpiry:    defaultRefreshExpiry,
	}
}

// Execute rotates an existing refresh token and returns a new token pair.
func (u *TokenRefreshUseCaseImpl) Execute(ctx context.Context, input authtypes.TokenRefreshRequest, ipAddress, userAgent string) (*authtypes.TokenRefreshResponse, error) {
	rawToken := strings.TrimSpace(input.RefreshToken)
	if rawToken == "" {
		return nil, domainerrors.ErrInvalidToken
	}

	if _, err := u.tokenService.ValidateRefreshToken(rawToken); err != nil {
		return nil, domainerrors.ErrInvalidToken
	}

	tokenHash := u.hashService.HashToken(rawToken)

	storedToken, err := u.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil || storedToken == nil {
		return nil, domainerrors.ErrInvalidToken
	}

	// Token reuse detection: if a revoked token is presented, revoke all tokens in the family
	if storedToken.RevokedAt != nil {
		u.revokeFamilyOnReuse(ctx, storedToken.Family)
		return nil, domainerrors.ErrTokenRevoked
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return nil, domainerrors.ErrTokenExpired
	}

	// Revoke the old token upon rotation
	// The revoke is conditional/atomic: losing a race (RowsAffected == 0) means
	// the token was already used, so treat it as reuse and mint nothing.
	if err := u.refreshTokenRepo.Revoke(ctx, storedToken.ID); err != nil {
		if errors.Is(err, domainerrors.ErrTokenAlreadyRevoked) {
			u.revokeFamilyOnReuse(ctx, storedToken.Family)
			return nil, domainerrors.ErrTokenRevoked
		}
		return nil, err
	}

	user, err := u.userRepo.GetByIDWithRoles(ctx, storedToken.TenantID, storedToken.UserID)
	if err != nil || user == nil {
		return nil, domainerrors.ErrUserNotFound
	}

	if !user.IsActive {
		return nil, domainerrors.ErrUserInactive
	}

	roleNames := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleNames = append(roleNames, r.Name)
	}

	newRefreshTokenID := uuid.New()

	accessClaims := service.AccessTokenClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Roles:    roleNames,
		Email:    user.Email,
	}

	refreshClaims := service.RefreshTokenClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		TokenID:  newRefreshTokenID,
		Family:   storedToken.Family,
		Roles:    roleNames,
	}

	tokenPair, err := u.tokenService.GenerateTokenPair(accessClaims, refreshClaims)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	newTokenHash := u.hashService.HashToken(tokenPair.RefreshToken)

	newRT := &entity.RefreshToken{
		ID:        newRefreshTokenID,
		TenantID:  user.TenantID,
		UserID:    user.ID,
		TokenHash: newTokenHash,
		Family:    storedToken.Family,
		ExpiresAt: now.Add(u.refreshExpiry),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.refreshTokenRepo.Create(ctx, newRT); err != nil {
		return nil, err
	}

	return &authtypes.TokenRefreshResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		TokenType:    tokenPair.TokenType,
	}, nil
}

// revokeFamilyOnReuse revokes every token in the family after reuse detection.
// ErrTokenRevoked is returned to the caller regardless, but a failure to
// actually revoke the rest of the family must not vanish silently -- it leaves
// those tokens live.
func (u *TokenRefreshUseCaseImpl) revokeFamilyOnReuse(ctx context.Context, family uuid.UUID) {
	if err := u.refreshTokenRepo.RevokeFamily(ctx, family); err != nil && u.logger != nil {
		u.logger.Error(ctx, "token_refresh: failed to revoke token family on reuse detection", err)
	}
}
