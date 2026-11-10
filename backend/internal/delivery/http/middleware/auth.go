package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/ctx"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	authusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/auth"
)

// Gin context keys for authenticated user information.
const (
	ContextKeyUserID   = "user_id"
	ContextKeyTenantID = "tenant_id"
	ContextKeyRoles    = "roles"
	ContextKeyEmail    = "email"
	ContextKeyClaims   = "claims"
)

// Auth validates the Bearer access token from the Authorization header using
// the provided TokenService and injects the authenticated identity (user ID,
// tenant ID, roles, email) into both the Gin context and the request's Go context.
//
// Requests with missing, malformed, expired, or invalid tokens are aborted with
// a 401 Unauthorized response before reaching downstream handlers.
//
// After the token validates, the verifier confirms against the database that the
// token's tenant and user still exist and are usable (tenant active and not
// soft-deleted; user in that tenant, active and not soft-deleted). An unusable
// identity yields a uniform 401 that does not reveal which check failed; a
// verifier/DB failure fails closed with 503. The roles placed in the context
// are the user's current roles from the database, not the token's roles.
func Auth(tokenService domainservice.TokenService, verifier authusecase.VerifyIdentityUseCase) gin.HandlerFunc {
	if verifier == nil {
		panic("middleware.Auth: identity verifier is required")
	}
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			response.Unauthorized(c, "invalid authorization header format")
			c.Abort()
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		claims, err := tokenService.ValidateAccessToken(tokenStr)
		if err != nil {
			if errors.Is(err, domainerrors.ErrTokenExpired) {
				response.Unauthorized(c, "token has expired")
			} else {
				response.Unauthorized(c, "invalid or expired token")
			}
			c.Abort()
			return
		}

		if claims == nil || claims.UserID == uuid.Nil || claims.TenantID == uuid.Nil {
			response.Unauthorized(c, "invalid token claims")
			c.Abort()
			return
		}

		identity, err := verifier.Execute(c.Request.Context(), claims.TenantID, claims.UserID)
		if err != nil {
			if errors.Is(err, domainerrors.ErrUnauthorized) {
				response.Unauthorized(c, "invalid or expired token")
			} else {
				response.ServiceUnavailable(c, "unable to verify identity")
			}
			c.Abort()
			return
		}
		// Do not trust the token's roles: use the current ones from the database.
		claims.Roles = identity.Roles

		userIDStr := claims.UserID.String()
		tenantIDStr := claims.TenantID.String()

		// Set on Gin context
		c.Set(ContextKeyUserID, userIDStr)
		c.Set(ContextKeyTenantID, tenantIDStr)
		c.Set(ContextKeyRoles, claims.Roles)
		c.Set(ContextKeyEmail, claims.Email)
		c.Set(ContextKeyClaims, claims)

		// Inject into Go request context for usecase / repository layer
		reqCtx := c.Request.Context()
		reqCtx = ctx.WithUserID(reqCtx, userIDStr)
		reqCtx = ctx.WithTenantID(reqCtx, tenantIDStr)
		reqCtx = ctx.WithRoles(reqCtx, claims.Roles)
		c.Request = c.Request.WithContext(reqCtx)

		c.Next()
	}
}

// RequireRole creates a middleware that checks if the authenticated user possesses
// at least one of the allowed roles. If not, it aborts the request with 403 Forbidden.
// If no authenticated identity is present in the context at all, it aborts with
// 401 Unauthorized instead — an authenticated user with zero matching (or zero
// assigned) roles is a 403, not a 401.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := getUserID(c); err != nil {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		roles := getRoles(c)

		if HasAnyRole(roles, allowedRoles...) {
			c.Next()
			return
		}

		response.Forbidden(c, "insufficient permissions")
		c.Abort()
	}
}

// HasRole checks whether the role slice contains the target role.
func HasRole(userRoles []string, targetRole string) bool {
	for _, r := range userRoles {
		if r == targetRole {
			return true
		}
	}
	return false
}

// HasAnyRole checks whether the user has at least one of the target roles.
func HasAnyRole(userRoles []string, targetRoles ...string) bool {
	for _, target := range targetRoles {
		if HasRole(userRoles, target) {
			return true
		}
	}
	return false
}

// getUserID retrieves the authenticated user ID as a UUID from Gin context or request context.
func getUserID(c *gin.Context) (uuid.UUID, error) {
	if val, ok := c.Get(ContextKeyUserID); ok {
		if strVal, ok := val.(string); ok && strVal != "" {
			return uuid.Parse(strVal)
		}
		if uid, ok := val.(uuid.UUID); ok {
			return uid, nil
		}
	}
	if strVal, ok := ctx.UserIDFromContext(c.Request.Context()); ok && strVal != "" {
		return uuid.Parse(strVal)
	}
	return uuid.Nil, domainerrors.ErrUnauthorized
}

// getRoles retrieves the roles slice from Gin context or request context.
func getRoles(c *gin.Context) []string {
	if val, ok := c.Get(ContextKeyRoles); ok {
		if roles, ok := val.([]string); ok {
			return roles
		}
	}
	if roles, ok := ctx.RolesFromContext(c.Request.Context()); ok {
		return roles
	}
	return nil
}

// getClaims retrieves the parsed AccessTokenClaims from Gin context.
func getClaims(c *gin.Context) (*domainservice.AccessTokenClaims, bool) {
	if val, ok := c.Get(ContextKeyClaims); ok {
		if claims, ok := val.(*domainservice.AccessTokenClaims); ok {
			return claims, true
		}
	}
	return nil, false
}
