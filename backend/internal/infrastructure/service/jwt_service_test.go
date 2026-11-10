package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

func newTestJWTService(t *testing.T, secret string, accessExpiry, refreshExpiry time.Duration) domainservice.TokenService {
	t.Helper()
	svc, err := NewJWTService(secret, accessExpiry, refreshExpiry)
	if err != nil {
		t.Fatalf("unexpected error constructing jwt service: %v", err)
	}
	return svc
}

func TestNewJWTService_EmptySecret(t *testing.T) {
	_, err := NewJWTService("", 15*time.Minute, 7*24*time.Hour)
	if err == nil {
		t.Fatalf("expected error constructing jwt service with empty secret, got nil")
	}
}

func TestJWTService_AccessTokenFlow(t *testing.T) {
	jwtSvc := newTestJWTService(t, "test-secret-key-32-bytes-long!", 15*time.Minute, 7*24*time.Hour)

	userID := uuid.New()
	tenantID := uuid.New()
	claims := domainservice.AccessTokenClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    "test@example.com",
		Roles:    []string{"admin"},
	}

	tokenStr, expiresAt, err := jwtSvc.GenerateAccessToken(claims)
	if err != nil {
		t.Fatalf("unexpected error generating access token: %v", err)
	}

	if tokenStr == "" {
		t.Fatalf("generated token string should not be empty")
	}

	if expiresAt.Before(time.Now()) {
		t.Fatalf("expiration should be in the future")
	}

	parsedClaims, err := jwtSvc.ValidateAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}

	if parsedClaims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, parsedClaims.UserID)
	}
	if parsedClaims.TenantID != tenantID {
		t.Errorf("expected TenantID %s, got %s", tenantID, parsedClaims.TenantID)
	}
	if parsedClaims.Email != "test@example.com" {
		t.Errorf("expected Email test@example.com, got %s", parsedClaims.Email)
	}
	if len(parsedClaims.Roles) != 1 || parsedClaims.Roles[0] != "admin" {
		t.Errorf("expected roles [admin], got %v", parsedClaims.Roles)
	}
}

func TestJWTService_RefreshTokenFlow(t *testing.T) {
	jwtSvc := newTestJWTService(t, "test-secret-key-32-bytes-long!", 15*time.Minute, 7*24*time.Hour)

	userID := uuid.New()
	tenantID := uuid.New()
	tokenID := uuid.New()
	family := uuid.New()

	claims := domainservice.RefreshTokenClaims{
		UserID:   userID,
		TenantID: tenantID,
		TokenID:  tokenID,
		Family:   family,
		Roles:    []string{"employee"},
	}

	tokenStr, expiresAt, err := jwtSvc.GenerateRefreshToken(claims)
	if err != nil {
		t.Fatalf("unexpected error generating refresh token: %v", err)
	}

	if tokenStr == "" {
		t.Fatalf("generated token string should not be empty")
	}

	if expiresAt.Before(time.Now()) {
		t.Fatalf("expiration should be in the future")
	}

	parsedClaims, err := jwtSvc.ValidateRefreshToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}

	if parsedClaims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, parsedClaims.UserID)
	}
	if parsedClaims.TenantID != tenantID {
		t.Errorf("expected TenantID %s, got %s", tenantID, parsedClaims.TenantID)
	}
	if parsedClaims.TokenID != tokenID {
		t.Errorf("expected TokenID %s, got %s", tokenID, parsedClaims.TokenID)
	}
	if parsedClaims.Family != family {
		t.Errorf("expected Family %s, got %s", family, parsedClaims.Family)
	}
	if len(parsedClaims.Roles) != 1 || parsedClaims.Roles[0] != "employee" {
		t.Errorf("expected roles [employee], got %v", parsedClaims.Roles)
	}
}

func TestJWTService_GenerateTokenPair(t *testing.T) {
	jwtSvc := newTestJWTService(t, "test-secret-key-32-bytes-long!", 15*time.Minute, 7*24*time.Hour)

	userID := uuid.New()
	tenantID := uuid.New()

	accessClaims := domainservice.AccessTokenClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    "pair@example.com",
		Roles:    []string{"super_admin"},
	}

	refreshClaims := domainservice.RefreshTokenClaims{
		UserID:   userID,
		TenantID: tenantID,
		TokenID:  uuid.New(),
		Family:   uuid.New(),
		Roles:    []string{"super_admin"},
	}

	pair, err := jwtSvc.GenerateTokenPair(accessClaims, refreshClaims)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("tokens in pair should not be empty")
	}
	if pair.TokenType != "Bearer" {
		t.Errorf("expected token type Bearer, got %s", pair.TokenType)
	}

	validatedAccess, err := jwtSvc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("failed to validate access token from pair: %v", err)
	}
	if validatedAccess.Email != "pair@example.com" {
		t.Errorf("expected email pair@example.com, got %s", validatedAccess.Email)
	}

	validatedRefresh, err := jwtSvc.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("failed to validate refresh token from pair: %v", err)
	}
	if len(validatedRefresh.Roles) != 1 || validatedRefresh.Roles[0] != "super_admin" {
		t.Errorf("expected roles [super_admin], got %v", validatedRefresh.Roles)
	}
}

func TestJWTService_ExpiredToken(t *testing.T) {
	jwtSvc := newTestJWTService(t, "test-secret-key-32-bytes-long!", -1*time.Minute, -1*time.Minute)

	tokenStr, _, err := jwtSvc.GenerateAccessToken(domainservice.AccessTokenClaims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Email:    "expired@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = jwtSvc.ValidateAccessToken(tokenStr)
	if err != domainerrors.ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestJWTService_InvalidSignature(t *testing.T) {
	jwtSvc1 := newTestJWTService(t, "secret-1-32-bytes-long-padding!", 15*time.Minute, 7*24*time.Hour)
	jwtSvc2 := newTestJWTService(t, "secret-2-32-bytes-long-padding!", 15*time.Minute, 7*24*time.Hour)

	tokenStr, _, err := jwtSvc1.GenerateAccessToken(domainservice.AccessTokenClaims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Email:    "test@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = jwtSvc2.ValidateAccessToken(tokenStr)
	if err != domainerrors.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken on signature mismatch, got %v", err)
	}
}

func TestJWTService_WrongTokenTypeRejected(t *testing.T) {
	jwtSvc := newTestJWTService(t, "test-secret-key-32-bytes-long!", 15*time.Minute, 7*24*time.Hour)

	refreshTokenStr, _, err := jwtSvc.GenerateRefreshToken(domainservice.RefreshTokenClaims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		TokenID:  uuid.New(),
		Family:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := jwtSvc.ValidateAccessToken(refreshTokenStr); err != domainerrors.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken when validating a refresh token as an access token, got %v", err)
	}

	accessTokenStr, _, err := jwtSvc.GenerateAccessToken(domainservice.AccessTokenClaims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Email:    "test@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := jwtSvc.ValidateRefreshToken(accessTokenStr); err != domainerrors.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken when validating an access token as a refresh token, got %v", err)
	}
}
