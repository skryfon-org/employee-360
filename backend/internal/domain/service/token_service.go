package service

import (
	"time"

	"github.com/google/uuid"
)

// AccessTokenClaims represents the claims encoded in an access JWT token.
type AccessTokenClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Roles    []string  `json:"roles"`
	Email    string    `json:"email"`
}

// RefreshTokenClaims represents the claims encoded in a refresh JWT token.
type RefreshTokenClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	TokenID  uuid.UUID `json:"token_id"`
	Family   uuid.UUID `json:"family"`
	Roles    []string  `json:"roles"`
}

// TokenPair contains generated access and refresh tokens.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// TokenService defines operations for generating and validating access and refresh JWT tokens.
type TokenService interface {
	// GenerateAccessToken creates a signed access JWT for a user.
	GenerateAccessToken(claims AccessTokenClaims) (string, time.Time, error)

	// GenerateRefreshToken creates a signed refresh JWT for a user session.
	GenerateRefreshToken(claims RefreshTokenClaims) (string, time.Time, error)

	// GenerateTokenPair generates both an access and refresh token.
	GenerateTokenPair(accessClaims AccessTokenClaims, refreshClaims RefreshTokenClaims) (*TokenPair, error)

	// ValidateAccessToken validates and parses an access token string.
	ValidateAccessToken(tokenString string) (*AccessTokenClaims, error)

	// ValidateRefreshToken validates and parses a refresh token string.
	ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error)
}
