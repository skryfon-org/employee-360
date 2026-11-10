package auth

import (
	"context"
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

const (
	dummyBcryptHash      = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	defaultRefreshExpiry = 7 * 24 * time.Hour
)

// LoginUseCaseImpl implements authusecase.LoginUseCase.
type LoginUseCaseImpl struct {
	userRepo         repository.UserRepository
	tenantDomainRepo repository.TenantDomainRepository
	tokenService     service.TokenService
	hashService      service.HashService
	refreshTokenRepo repository.RefreshTokenRepository
	logger           service.Logger
	refreshExpiry    time.Duration
}

var _ authusecase.LoginUseCase = (*LoginUseCaseImpl)(nil)

// NewLoginUseCase constructs a new LoginUseCaseImpl.
func NewLoginUseCase(
	userRepo repository.UserRepository,
	tenantDomainRepo repository.TenantDomainRepository,
	tokenService service.TokenService,
	hashService service.HashService,
	refreshTokenRepo repository.RefreshTokenRepository,
	logger service.Logger,
) *LoginUseCaseImpl {
	return &LoginUseCaseImpl{
		userRepo:         userRepo,
		tenantDomainRepo: tenantDomainRepo,
		tokenService:     tokenService,
		hashService:      hashService,
		refreshTokenRepo: refreshTokenRepo,
		logger:           logger,
		refreshExpiry:    defaultRefreshExpiry,
	}
}

// Execute authenticates a user by email and password and returns a token pair.
// ipAddress and userAgent are server-derived HTTP request metadata recorded on the
// issued refresh token for audit purposes.
func (u *LoginUseCaseImpl) Execute(ctx context.Context, req authtypes.LoginRequest, ipAddress, userAgent string) (*authtypes.LoginResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	password := req.Password

	if email == "" || password == "" {
		_ = u.hashService.ComparePassword(dummyBcryptHash, "dummy")
		return nil, domainerrors.ErrInvalidCredentials
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		_ = u.hashService.ComparePassword(dummyBcryptHash, "dummy")
		return nil, domainerrors.ErrInvalidCredentials
	}

	// Resolve the tenant server-side from the email's domain (tenant_domains),
	// then look the user up scoped to that tenant. The same email may exist in
	// several tenants, so a cross-tenant lookup by email alone is ambiguous.
	tenant, err := u.tenantDomainRepo.FindTenantByDomain(ctx, parts[1])
	if err != nil || tenant == nil {
		_ = u.hashService.ComparePassword(dummyBcryptHash, password)
		return nil, domainerrors.ErrInvalidCredentials
	}

	user, err := u.userRepo.GetByTenantAndEmailWithRoles(ctx, tenant.ID, email)
	if err != nil || user == nil {
		_ = u.hashService.ComparePassword(dummyBcryptHash, password)
		return nil, domainerrors.ErrInvalidCredentials
	}

	if user.PasswordHash == nil || *user.PasswordHash == "" {
		_ = u.hashService.ComparePassword(dummyBcryptHash, password)
		return nil, domainerrors.ErrInvalidCredentials
	}

	if err := u.hashService.ComparePassword(*user.PasswordHash, password); err != nil {
		return nil, domainerrors.ErrInvalidCredentials
	}

	// IsActive is checked only after the password has been verified, so an
	// invalid password always yields ErrInvalidCredentials regardless of
	// account status — never leaking account existence/state to an attacker.
	if !user.IsActive {
		return nil, domainerrors.ErrUserInactive
	}

	roleNames := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleNames = append(roleNames, r.Name)
	}

	refreshTokenID := uuid.New()
	family := uuid.New()

	accessClaims := service.AccessTokenClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Roles:    roleNames,
		Email:    user.Email,
	}

	refreshClaims := service.RefreshTokenClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		TokenID:  refreshTokenID,
		Family:   family,
		Roles:    roleNames,
	}

	tokenPair, err := u.tokenService.GenerateTokenPair(accessClaims, refreshClaims)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	expiry := now.Add(u.refreshExpiry)
	tokenHash := u.hashService.HashToken(tokenPair.RefreshToken)

	rt := &entity.RefreshToken{
		ID:        refreshTokenID,
		TenantID:  user.TenantID,
		UserID:    user.ID,
		TokenHash: tokenHash,
		Family:    family,
		ExpiresAt: expiry,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.refreshTokenRepo.Create(ctx, rt); err != nil {
		return nil, err
	}

	user.LastLoginAt = &now
	if err := u.userRepo.Update(ctx, user.TenantID, user); err != nil && u.logger != nil {
		// LastLoginAt is a best-effort side effect, not the primary security
		// control, so a failure here must not fail the login -- but it must
		// not vanish silently either.
		u.logger.Error(ctx, "login: failed to update user.LastLoginAt", err)
	}

	return &authtypes.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		TokenType:    tokenPair.TokenType,
		User:         user,
	}, nil
}
