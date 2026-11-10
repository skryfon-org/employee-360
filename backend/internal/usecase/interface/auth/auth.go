package auth

import (
	"context"

	"github.com/google/uuid"
	authtypes "github.com/skryfon-org/employee-360/backend/internal/types/auth"
)

// LoginUseCase defines the port for user login across all roles.
//
// ipAddress and userAgent are passed as explicit parameters rather than fields
// on LoginRequest: they are server-derived HTTP request metadata used only for
// refresh-token audit fields -- neither is part of what the client actually
// submits to log in (see authtypes.LoginRequest).
type LoginUseCase interface {
	Execute(ctx context.Context, req authtypes.LoginRequest, ipAddress, userAgent string) (*authtypes.LoginResponse, error)
}

// TokenRefreshUseCase defines the port for rotating refresh tokens.
//
// ipAddress and userAgent are server-derived HTTP request metadata (never part
// of the client payload), recorded on the rotated refresh token for audit.
type TokenRefreshUseCase interface {
	Execute(ctx context.Context, req authtypes.TokenRefreshRequest, ipAddress, userAgent string) (*authtypes.TokenRefreshResponse, error)
}

// LogoutUseCase defines the port for revoking sessions and refresh tokens.
//
// tenantID and userID are the authenticated caller's identity from the verified
// access token; the presented refresh token must belong to them.
type LogoutUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID, req authtypes.LogoutRequest) error
}

// ForgotPasswordUseCase defines the port for requesting a password reset (enumeration-safe).
//
// The tenant is resolved inside the usecase from the email's domain
// (tenant_domains); it is never client-supplied.
type ForgotPasswordUseCase interface {
	Execute(ctx context.Context, req authtypes.ForgotPasswordRequest) error
}

// ResetPasswordUseCase defines the port for executing a password reset with a valid token.
type ResetPasswordUseCase interface {
	Execute(ctx context.Context, req authtypes.ResetPasswordRequest) error
}

// VerifyIdentityUseCase defines the port used by the auth middleware to confirm
// that the tenant_id and user_id from a validated access token still refer to a
// usable tenant and user, and to load the user's current roles.
//
// Execute returns domainerrors.ErrUnauthorized when the identity is not usable
// (tenant missing, soft-deleted or inactive; user missing in that tenant,
// soft-deleted or inactive) -- deliberately one error so callers cannot leak
// which check failed. Any other error is an infrastructure failure and callers
// must fail closed.
type VerifyIdentityUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID) (*authtypes.VerifiedIdentity, error)
}
