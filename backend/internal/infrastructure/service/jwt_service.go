package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
	defaultIssuer    = "employee360"
)

// accessTokenClaims is the JWT claim set embedded in a signed access token.
type accessTokenClaims struct {
	UserID    string   `json:"user_id"`
	TenantID  string   `json:"tenant_id"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	TokenType string   `json:"token_type"`
	jwt.RegisteredClaims
}

// refreshTokenClaims is the JWT claim set embedded in a signed refresh token.
type refreshTokenClaims struct {
	UserID    string   `json:"user_id"`
	TenantID  string   `json:"tenant_id"`
	TokenID   string   `json:"token_id"`
	Family    string   `json:"family"`
	Roles     []string `json:"roles,omitempty"`
	TokenType string   `json:"token_type"`
	jwt.RegisteredClaims
}

type jwtService struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// NewJWTService creates a new JWT-based TokenService implementation backed by
// github.com/golang-jwt/jwt/v5. It returns an error if secret is empty, since
// an empty/misconfigured signing secret would otherwise silently produce
// weak, predictable signatures instead of failing fast at construction.
func NewJWTService(secret string, accessExpiry, refreshExpiry time.Duration) (domainservice.TokenService, error) {
	if secret == "" {
		return nil, errors.New("jwt: secret must not be empty")
	}

	return &jwtService{
		secret:        []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        defaultIssuer,
	}, nil
}

// GenerateAccessToken generates a signed access JWT for a user.
func (s *jwtService) GenerateAccessToken(claims domainservice.AccessTokenClaims) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.accessExpiry)

	c := accessTokenClaims{
		UserID:    claims.UserID.String(),
		TenantID:  claims.TenantID.String(),
		Email:     claims.Email,
		Roles:     claims.Roles,
		TokenType: tokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenStr, expiresAt, nil
}

// GenerateRefreshToken generates a signed refresh JWT for a session.
func (s *jwtService) GenerateRefreshToken(claims domainservice.RefreshTokenClaims) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.refreshExpiry)

	c := refreshTokenClaims{
		UserID:    claims.UserID.String(),
		TenantID:  claims.TenantID.String(),
		TokenID:   claims.TokenID.String(),
		Family:    claims.Family.String(),
		Roles:     claims.Roles,
		TokenType: tokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenStr, expiresAt, nil
}

// GenerateTokenPair generates both an access and refresh token.
func (s *jwtService) GenerateTokenPair(accessClaims domainservice.AccessTokenClaims, refreshClaims domainservice.RefreshTokenClaims) (*domainservice.TokenPair, error) {
	accessToken, accessExpiresAt, err := s.GenerateAccessToken(accessClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, _, err := s.GenerateRefreshToken(refreshClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &domainservice.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt,
		TokenType:    "Bearer",
	}, nil
}

// ValidateAccessToken parses and validates a signed access JWT string.
func (s *jwtService) ValidateAccessToken(tokenString string) (*domainservice.AccessTokenClaims, error) {
	claims := &accessTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, s.keyFunc, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		return nil, mapJWTError(err)
	}
	if !token.Valid {
		return nil, domainerrors.ErrInvalidToken
	}

	if claims.TokenType != tokenTypeAccess {
		return nil, domainerrors.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, domainerrors.ErrInvalidToken
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return nil, domainerrors.ErrInvalidToken
	}

	return &domainservice.AccessTokenClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    claims.Email,
		Roles:    claims.Roles,
	}, nil
}

// ValidateRefreshToken parses and validates a signed refresh JWT string.
func (s *jwtService) ValidateRefreshToken(tokenString string) (*domainservice.RefreshTokenClaims, error) {
	claims := &refreshTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, s.keyFunc, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		return nil, mapJWTError(err)
	}
	if !token.Valid {
		return nil, domainerrors.ErrInvalidToken
	}

	if claims.TokenType != tokenTypeRefresh {
		return nil, domainerrors.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, domainerrors.ErrInvalidToken
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return nil, domainerrors.ErrInvalidToken
	}

	tokenID, err := uuid.Parse(claims.TokenID)
	if err != nil {
		return nil, domainerrors.ErrInvalidToken
	}

	family, err := uuid.Parse(claims.Family)
	if err != nil {
		return nil, domainerrors.ErrInvalidToken
	}

	return &domainservice.RefreshTokenClaims{
		UserID:   userID,
		TenantID: tenantID,
		TokenID:  tokenID,
		Family:   family,
		Roles:    claims.Roles,
	}, nil
}

// keyFunc supplies the HMAC signing secret to jwt.ParseWithClaims.
func (s *jwtService) keyFunc(_ *jwt.Token) (interface{}, error) {
	return s.secret, nil
}

// mapJWTError translates golang-jwt parse/validation errors into this
// project's domain sentinel errors.
func mapJWTError(err error) error {
	if errors.Is(err, jwt.ErrTokenExpired) {
		return domainerrors.ErrTokenExpired
	}
	return domainerrors.ErrInvalidToken
}
