package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	authtypes "github.com/skryfon-org/employee-360/backend/internal/types/auth"
)

// tenantEmailKey composite-keys a user by (tenantID, email) so the mock can
// actually distinguish users that share an email across different tenants.
type tenantEmailKey struct {
	tenantID uuid.UUID
	email    string
}

// MockUserRepository implements repository.UserRepository for testing.
type mockUserRepository struct {
	usersByID          map[uuid.UUID]*entity.User
	usersByTenantEmail map[tenantEmailKey]*entity.User
	updatedUsers       []*entity.User
	updateErr          error // when set, Update returns this error instead of succeeding
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		usersByID:          make(map[uuid.UUID]*entity.User),
		usersByTenantEmail: make(map[tenantEmailKey]*entity.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *entity.User) error {
	m.usersByID[user.ID] = user
	m.usersByTenantEmail[tenantEmailKey{tenantID: user.TenantID, email: strings.ToLower(user.Email)}] = user
	return nil
}

// GetByID enforces the tenantID param against the stored user's own tenant --
// a mismatch is treated exactly like "not found", mirroring the real
// GORM adapter's "id = ? AND tenant_id = ?" scoping, so a test that passes
// the wrong tenant here is actually caught rather than silently ignored.
func (m *mockUserRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.User, error) {
	u, ok := m.usersByID[id]
	if !ok || u.TenantID != tenantID {
		return nil, domainerrors.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepository) GetByTenantAndEmail(ctx context.Context, tenantID uuid.UUID, email string) (*entity.User, error) {
	u, ok := m.usersByTenantEmail[tenantEmailKey{tenantID: tenantID, email: email}]
	if !ok {
		return nil, domainerrors.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepository) GetByIDWithRoles(ctx context.Context, tenantID, id uuid.UUID) (*entity.User, error) {
	return m.GetByID(ctx, tenantID, id)
}

func (m *mockUserRepository) GetByTenantAndEmailWithRoles(ctx context.Context, tenantID uuid.UUID, email string) (*entity.User, error) {
	return m.GetByTenantAndEmail(ctx, tenantID, email)
}

// Update enforces the tenantID param against the user's own TenantID -- a
// mismatch is rejected (not found) rather than silently applied, mirroring
// the real GORM adapter's "id = ? AND tenant_id = ?" scoping.
func (m *mockUserRepository) Update(ctx context.Context, tenantID uuid.UUID, user *entity.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	existing, ok := m.usersByID[user.ID]
	if !ok || existing.TenantID != tenantID || user.TenantID != tenantID {
		return domainerrors.ErrUserNotFound
	}
	m.updatedUsers = append(m.updatedUsers, user)
	m.usersByID[user.ID] = user
	m.usersByTenantEmail[tenantEmailKey{tenantID: user.TenantID, email: user.Email}] = user
	return nil
}

// Delete enforces the tenantID param against the stored user's own tenant --
// a mismatch leaves the row untouched and returns not found, mirroring the
// real GORM adapter's "id = ? AND tenant_id = ?" scoping.
func (m *mockUserRepository) Delete(ctx context.Context, tenantID, id, actorID uuid.UUID) error {
	u, ok := m.usersByID[id]
	if !ok || u.TenantID != tenantID {
		return domainerrors.ErrUserNotFound
	}
	delete(m.usersByID, id)
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*entity.User, int64, error) {
	var list []*entity.User
	for _, u := range m.usersByID {
		if u.TenantID == tenantID {
			list = append(list, u)
		}
	}
	return list, int64(len(list)), nil
}

// mockTenantDomainRepository implements repository.TenantDomainRepository.
type mockTenantDomainRepository struct {
	tenantsByDomain map[string]*entity.Tenant
	err             error
}

func newMockTenantDomainRepository() *mockTenantDomainRepository {
	return &mockTenantDomainRepository{tenantsByDomain: make(map[string]*entity.Tenant)}
}

// with registers domain -> tenantID and returns the receiver for chaining.
func (m *mockTenantDomainRepository) with(domain string, tenantID uuid.UUID) *mockTenantDomainRepository {
	m.tenantsByDomain[strings.ToLower(domain)] = &entity.Tenant{ID: tenantID, IsActive: true}
	return m
}

func (m *mockTenantDomainRepository) DomainBelongsToTenant(_ context.Context, tenantID uuid.UUID, domain string) (bool, error) {
	t, ok := m.tenantsByDomain[domain]
	return ok && t.ID == tenantID, nil
}

func (m *mockTenantDomainRepository) FindTenantByDomain(ctx context.Context, domain string) (*entity.Tenant, error) {
	if m.err != nil {
		return nil, m.err
	}
	t, ok := m.tenantsByDomain[strings.ToLower(domain)]
	if !ok {
		return nil, domainerrors.ErrTenantNotFound
	}
	return t, nil
}

// MockRefreshTokenRepository implements repository.RefreshTokenRepository.
type mockRefreshTokenRepository struct {
	tokensByHash    map[string]*entity.RefreshToken
	tokensByID      map[uuid.UUID]*entity.RefreshToken
	revokedTokenIDs []uuid.UUID
	revokedFamilies []uuid.UUID
	revokedUserIDs  []uuid.UUID
	revokeFamilyErr error // when set, RevokeFamily returns this error instead of succeeding
}

func newMockRefreshTokenRepository() *mockRefreshTokenRepository {
	return &mockRefreshTokenRepository{
		tokensByHash: make(map[string]*entity.RefreshToken),
		tokensByID:   make(map[uuid.UUID]*entity.RefreshToken),
	}
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	m.tokensByHash[token.TokenHash] = token
	m.tokensByID[token.ID] = token
	return nil
}

func (m *mockRefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	t, ok := m.tokensByHash[tokenHash]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return t, nil
}

func (m *mockRefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	m.revokedTokenIDs = append(m.revokedTokenIDs, id)
	if t, ok := m.tokensByID[id]; ok {
		if t.RevokedAt != nil {
			return domainerrors.ErrTokenAlreadyRevoked
		}
		now := time.Now().UTC()
		t.RevokedAt = &now
	}
	return nil
}

func (m *mockRefreshTokenRepository) RevokeFamily(ctx context.Context, family uuid.UUID) error {
	if m.revokeFamilyErr != nil {
		return m.revokeFamilyErr
	}
	m.revokedFamilies = append(m.revokedFamilies, family)
	now := time.Now().UTC()
	for _, t := range m.tokensByID {
		if t.Family == family {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	m.revokedUserIDs = append(m.revokedUserIDs, userID)
	now := time.Now().UTC()
	for _, t := range m.tokensByID {
		if t.UserID == userID {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockRefreshTokenRepository) DeleteExpiredTokens(ctx context.Context, before time.Time) error {
	return nil
}

// MockHashService implements service.HashService.
type mockHashService struct {
	compareErr error
}

func (m *mockHashService) HashPassword(password string) (string, error) {
	return "hashed_" + password, nil
}

func (m *mockHashService) ComparePassword(hashedPassword, password string) error {
	if m.compareErr != nil {
		return m.compareErr
	}
	if hashedPassword != "hashed_"+password {
		return domainerrors.ErrInvalidCredentials
	}
	return nil
}

func (m *mockHashService) HashToken(plainToken string) string {
	return "sha256_" + plainToken
}

// MockTokenService implements service.TokenService.
type mockTokenService struct {
	validateAccessClaims  *domainservice.AccessTokenClaims
	validateRefreshClaims *domainservice.RefreshTokenClaims
	validateErr           error
}

func (m *mockTokenService) GenerateAccessToken(claims domainservice.AccessTokenClaims) (string, time.Time, error) {
	return "access_jwt_" + claims.Email, time.Now().Add(15 * time.Minute), nil
}

func (m *mockTokenService) GenerateRefreshToken(claims domainservice.RefreshTokenClaims) (string, time.Time, error) {
	return "refresh_jwt_" + claims.TokenID.String(), time.Now().Add(7 * 24 * time.Hour), nil
}

func (m *mockTokenService) GenerateTokenPair(accessClaims domainservice.AccessTokenClaims, refreshClaims domainservice.RefreshTokenClaims) (*domainservice.TokenPair, error) {
	acc, exp, _ := m.GenerateAccessToken(accessClaims)
	ref, _, _ := m.GenerateRefreshToken(refreshClaims)
	return &domainservice.TokenPair{
		AccessToken:  acc,
		RefreshToken: ref,
		ExpiresAt:    exp,
		TokenType:    "Bearer",
	}, nil
}

func (m *mockTokenService) ValidateAccessToken(tokenString string) (*domainservice.AccessTokenClaims, error) {
	if m.validateErr != nil {
		return nil, m.validateErr
	}
	return m.validateAccessClaims, nil
}

func (m *mockTokenService) ValidateRefreshToken(tokenString string) (*domainservice.RefreshTokenClaims, error) {
	if m.validateErr != nil {
		return nil, m.validateErr
	}
	return m.validateRefreshClaims, nil
}

func TestLoginUseCase_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	pwHash := "hashed_Secret123!"

	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{}

	user := &entity.User{
		ID:           userID,
		TenantID:     tenantID,
		Email:        "alice@example.com",
		PasswordHash: &pwHash,
		IsActive:     true,
		Roles: []entity.Role{
			{ID: uuid.New(), TenantID: tenantID, Name: entity.RoleAdmin},
		},
	}
	_ = userRepo.Create(context.Background(), user)

	tdRepo := newMockTenantDomainRepository().with("example.com", tenantID)
	uc := NewLoginUseCase(userRepo, tdRepo, tokenSvc, hashSvc, refreshTokenRepo, &mockLogger{})

	output, err := uc.Execute(context.Background(), authtypes.LoginRequest{
		Email:    "alice@example.com",
		Password: "Secret123!",
	}, "127.0.0.1", "TestBrowser/1.0")

	if err != nil {
		t.Fatalf("expected successful login, got error: %v", err)
	}

	if output.AccessToken == "" || output.RefreshToken == "" {
		t.Fatalf("expected non-empty access and refresh tokens")
	}

	if output.User.ID != userID {
		t.Errorf("expected user ID %s, got %s", userID, output.User.ID)
	}

	// Verify refresh token was hashed and stored
	hashedRefToken := hashSvc.HashToken(output.RefreshToken)
	storedRT, ok := refreshTokenRepo.tokensByHash[hashedRefToken]
	if !ok {
		t.Fatalf("expected refresh token to be hashed and persisted in repo")
	}
	if storedRT.UserID != userID {
		t.Errorf("expected stored token user ID %s, got %s", userID, storedRT.UserID)
	}
	if storedRT.TenantID != tenantID {
		t.Errorf("expected stored token tenant ID %s, got %s", tenantID, storedRT.TenantID)
	}

	// Verify last login was updated
	if len(userRepo.updatedUsers) == 0 || userRepo.updatedUsers[0].LastLoginAt == nil {
		t.Errorf("expected user LastLoginAt to be updated")
	}
}

func TestLoginUseCase_InvalidCredentials(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	pwHash := "hashed_Secret123!"

	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{}

	user := &entity.User{
		ID:           userID,
		TenantID:     tenantID,
		Email:        "alice@example.com",
		PasswordHash: &pwHash,
		IsActive:     true,
	}
	_ = userRepo.Create(context.Background(), user)

	tdRepo := newMockTenantDomainRepository().with("example.com", tenantID)
	uc := NewLoginUseCase(userRepo, tdRepo, tokenSvc, hashSvc, refreshTokenRepo, &mockLogger{})

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"wrong password", "alice@example.com", "WrongPassword!"},
		{"unknown user", "bob@example.com", "Secret123!"},
		{"empty email", "", "Secret123!"},
		{"empty password", "alice@example.com", ""},
		{"invalid email format", "not-an-email", "Secret123!"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.Execute(context.Background(), authtypes.LoginRequest{
				Email:    tc.email,
				Password: tc.password,
			}, "", "")
			if err != domainerrors.ErrInvalidCredentials {
				t.Errorf("expected ErrInvalidCredentials, got %v", err)
			}
		})
	}
}

func TestLoginUseCase_InactiveUser(t *testing.T) {
	tenantID := uuid.New()
	pwHash := "hashed_Secret123!"

	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{}

	user := &entity.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        "inactive@example.com",
		PasswordHash: &pwHash,
		IsActive:     false,
	}
	_ = userRepo.Create(context.Background(), user)

	tdRepo := newMockTenantDomainRepository().with("example.com", tenantID)
	uc := NewLoginUseCase(userRepo, tdRepo, tokenSvc, hashSvc, refreshTokenRepo, &mockLogger{})

	_, err := uc.Execute(context.Background(), authtypes.LoginRequest{
		Email:    "inactive@example.com",
		Password: "Secret123!",
	}, "", "")

	if err != domainerrors.ErrUserInactive {
		t.Errorf("expected ErrUserInactive, got %v", err)
	}
}

// TestLoginUseCase_InactiveUser_WrongPassword proves that an incorrect
// password for an inactive user's email still yields ErrInvalidCredentials
// (not ErrUserInactive) — the account's active/inactive state must never be
// distinguishable from "unknown user" or "wrong password" before the
// password has actually been verified, to avoid a user-enumeration leak.
func TestLoginUseCase_InactiveUser_WrongPassword(t *testing.T) {
	tenantID := uuid.New()
	pwHash := "hashed_Secret123!"

	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{}

	user := &entity.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        "inactive@example.com",
		PasswordHash: &pwHash,
		IsActive:     false,
	}
	_ = userRepo.Create(context.Background(), user)

	tdRepo := newMockTenantDomainRepository().with("example.com", tenantID)
	uc := NewLoginUseCase(userRepo, tdRepo, tokenSvc, hashSvc, refreshTokenRepo, &mockLogger{})

	_, err := uc.Execute(context.Background(), authtypes.LoginRequest{
		Email:    "inactive@example.com",
		Password: "WrongPassword!",
	}, "", "")

	if err != domainerrors.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for a wrong password against an inactive account, got %v", err)
	}
}

// TestLoginUseCase_ResolvesUserTenant proves that the tenant is resolved from
// the email's domain (tenant_domains) and that the same email address existing
// in two tenants never yields the wrong tenant's user.
func TestLoginUseCase_ResolvesUserTenant(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()
	pwHashA := "hashed_PasswordForA!"
	pwHashB := "hashed_PasswordForB!"

	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{}
	tdRepo := newMockTenantDomainRepository().with("a.example.com", tenantA).with("b.example.com", tenantB)

	_ = userRepo.Create(context.Background(), &entity.User{
		ID: userA, TenantID: tenantA, Email: "user@a.example.com", PasswordHash: &pwHashA, IsActive: true,
	})
	_ = userRepo.Create(context.Background(), &entity.User{
		ID: userB, TenantID: tenantB, Email: "user@b.example.com", PasswordHash: &pwHashB, IsActive: true,
	})

	uc := NewLoginUseCase(userRepo, tdRepo, tokenSvc, hashSvc, refreshTokenRepo, &mockLogger{})

	outputA, err := uc.Execute(context.Background(), authtypes.LoginRequest{
		Email: "user@a.example.com", Password: "PasswordForA!",
	}, "", "")
	if err != nil {
		t.Fatalf("expected userA login to succeed, got error: %v", err)
	}
	if outputA.User.ID != userA || outputA.User.TenantID != tenantA {
		t.Errorf("expected userA %s with tenant %s, got %s / %s", userA, tenantA, outputA.User.ID, outputA.User.TenantID)
	}

	outputB, err := uc.Execute(context.Background(), authtypes.LoginRequest{
		Email: "user@b.example.com", Password: "PasswordForB!",
	}, "", "")
	if err != nil {
		t.Fatalf("expected userB login to succeed, got error: %v", err)
	}
	if outputB.User.ID != userB || outputB.User.TenantID != tenantB {
		t.Errorf("expected userB %s with tenant %s, got %s / %s", userB, tenantB, outputB.User.ID, outputB.User.TenantID)
	}
}

// TestLoginUseCase_SameEmailInTwoTenants proves that when the identical email
// exists in two tenants, login resolves the tenant owning the email's domain
// and authenticates against that tenant's user only.
func TestLoginUseCase_SameEmailInTwoTenants(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()
	pwHashA := "hashed_PasswordForA!"
	pwHashB := "hashed_PasswordForB!"
	const sharedEmail = "shared@example.com"

	userRepo := newMockUserRepository()
	// Only tenantB owns example.com; tenantA has a same-email user but no claim on the domain.
	tdRepo := newMockTenantDomainRepository().with("example.com", tenantB)

	_ = userRepo.Create(context.Background(), &entity.User{
		ID: userA, TenantID: tenantA, Email: sharedEmail, PasswordHash: &pwHashA, IsActive: true,
	})
	_ = userRepo.Create(context.Background(), &entity.User{
		ID: userB, TenantID: tenantB, Email: sharedEmail, PasswordHash: &pwHashB, IsActive: true,
	})

	uc := NewLoginUseCase(userRepo, tdRepo, &mockTokenService{}, &mockHashService{}, newMockRefreshTokenRepository(), &mockLogger{})

	out, err := uc.Execute(context.Background(), authtypes.LoginRequest{Email: sharedEmail, Password: "PasswordForB!"}, "", "")
	if err != nil {
		t.Fatalf("expected tenantB user login to succeed, got: %v", err)
	}
	if out.User.ID != userB || out.User.TenantID != tenantB {
		t.Errorf("expected tenantB user %s, got %s / tenant %s", userB, out.User.ID, out.User.TenantID)
	}

	// tenantA's password must not authenticate: the domain resolves to tenantB.
	_, err = uc.Execute(context.Background(), authtypes.LoginRequest{Email: sharedEmail, Password: "PasswordForA!"}, "", "")
	if err != domainerrors.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for other tenant's password, got %v", err)
	}
}

// TestLoginUseCase_UnknownDomain proves a domain with no tenant_domains row
// yields ErrInvalidCredentials (no account-existence leak).
func TestLoginUseCase_UnknownDomain(t *testing.T) {
	tenantID := uuid.New()
	pwHash := "hashed_Secret123!"
	userRepo := newMockUserRepository()
	_ = userRepo.Create(context.Background(), &entity.User{
		ID: uuid.New(), TenantID: tenantID, Email: "alice@unmapped.example", PasswordHash: &pwHash, IsActive: true,
	})
	tdRepo := newMockTenantDomainRepository() // no domains registered

	uc := NewLoginUseCase(userRepo, tdRepo, &mockTokenService{}, &mockHashService{}, newMockRefreshTokenRepository(), &mockLogger{})

	_, err := uc.Execute(context.Background(), authtypes.LoginRequest{Email: "alice@unmapped.example", Password: "Secret123!"}, "", "")
	if err != domainerrors.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for unknown domain, got %v", err)
	}
}

// TestLoginUseCase_LastLoginUpdateFailureIsLogged proves that when persisting
// LastLoginAt fails, Execute still returns a successful login (it's a
// best-effort side effect, not the primary security control) but the failure
// is recorded via the Logger port rather than being silently discarded.
func TestLoginUseCase_LastLoginUpdateFailureIsLogged(t *testing.T) {
	tenantID := uuid.New()
	pwHash := "hashed_Secret123!"

	userRepo := newMockUserRepository()
	refreshTokenRepo := newMockRefreshTokenRepository()
	hashSvc := &mockHashService{}
	tokenSvc := &mockTokenService{}
	logger := &mockLogger{}

	updateErr := domainerrors.ErrUserNotFound // reused only as a sentinel error for this test
	user := &entity.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        "alice@example.com",
		PasswordHash: &pwHash,
		IsActive:     true,
	}
	_ = userRepo.Create(context.Background(), user)
	userRepo.updateErr = updateErr

	tdRepo := newMockTenantDomainRepository().with("example.com", tenantID)
	uc := NewLoginUseCase(userRepo, tdRepo, tokenSvc, hashSvc, refreshTokenRepo, logger)

	output, err := uc.Execute(context.Background(), authtypes.LoginRequest{
		Email:    "alice@example.com",
		Password: "Secret123!",
	}, "", "")

	if err != nil {
		t.Fatalf("expected login to succeed even when LastLoginAt update fails, got: %v", err)
	}
	if output == nil {
		t.Fatalf("expected a non-nil login output")
	}

	if len(logger.errorCalls) != 1 {
		t.Fatalf("expected the update failure to be logged exactly once, got %d calls", len(logger.errorCalls))
	}
	if logger.errorCalls[0].err != updateErr {
		t.Errorf("expected logged error to be the repository's error, got %v", logger.errorCalls[0].err)
	}
}
