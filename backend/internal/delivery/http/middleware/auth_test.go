package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/ctx"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
)

// setupTestJWTService returns a real JWT service wrapped so that tokens it mints
// register a healthy identity in a fake verifier (see identity_fake_test.go).
func setupTestJWTService(t *testing.T) domainservice.TokenService {
	t.Helper()
	svc, err := infraservice.NewJWTService("super-secret-test-key-32bytes-long!", 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create JWT service: %v", err)
	}
	return &registeringTokenService{TokenService: svc, verifier: newFakeVerifier()}
}

// verifierFor returns the fake verifier attached to a service from setupTestJWTService.
func verifierFor(tokenSvc domainservice.TokenService) *fakeVerifier {
	if r, ok := tokenSvc.(*registeringTokenService); ok {
		return r.verifier
	}
	return newFakeVerifier()
}

func setupAuthTestEngine(tokenSvc domainservice.TokenService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	authGroup := engine.Group("/protected", Auth(tokenSvc, verifierFor(tokenSvc)))
	{
		authGroup.GET("/me", func(c *gin.Context) {
			userID, _ := getUserID(c)
			roles := getRoles(c)
			claims, _ := getClaims(c)

			ctxUserID, _ := ctx.UserIDFromContext(c.Request.Context())
			ctxTenantID, _ := ctx.TenantIDFromContext(c.Request.Context())
			ctxRoles, _ := ctx.RolesFromContext(c.Request.Context())

			c.JSON(http.StatusOK, gin.H{
				"gin_user_id":   userID.String(),
				"gin_roles":     roles,
				"gin_email":     c.GetString(ContextKeyEmail),
				"claims_email":  claims.Email,
				"ctx_user_id":   ctxUserID,
				"ctx_tenant_id": ctxTenantID,
				"ctx_roles":     ctxRoles,
			})
		})

		authGroup.GET("/admin-only", RequireRole(entity.RoleAdmin, entity.RoleSuperAdmin), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "admin_granted"})
		})

		authGroup.GET("/super-admin-only", RequireRole(entity.RoleSuperAdmin), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "super_admin_granted"})
		})

		authGroup.GET("/employee-only", RequireRole(entity.RoleEmployee), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "employee_granted"})
		})
	}

	return engine
}

func TestAuth_MissingHeader(t *testing.T) {
	jwtSvc := setupTestJWTService(t)
	engine := setupAuthTestEngine(jwtSvc)

	req := httptest.NewRequest(http.MethodGet, "/protected/me", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized, got %d", rec.Code)
	}

	var resp response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Success {
		t.Errorf("expected success to be false")
	}
	if resp.Error == nil || resp.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected error code UNAUTHORIZED, got %v", resp.Error)
	}
}

func TestAuth_InvalidHeaderFormat(t *testing.T) {
	jwtSvc := setupTestJWTService(t)
	engine := setupAuthTestEngine(jwtSvc)

	testCases := []struct {
		name   string
		header string
	}{
		{"Basic auth scheme", "Basic dXNlcjpwYXNz"},
		{"No token after Bearer", "Bearer "},
		{"Multiple spaces only", "Bearer    "},
		{"Only Bearer word", "Bearer"},
		{"Custom prefix", "Token abc123xyz"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected/me", nil)
			req.Header.Set("Authorization", tc.header)
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected status 401, got %d", rec.Code)
			}

			var resp response.Envelope
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if resp.Success {
				t.Errorf("expected success to be false")
			}
		})
	}
}

func TestAuth_InvalidTokenSignature(t *testing.T) {
	jwtSvc := setupTestJWTService(t)
	engine := setupAuthTestEngine(jwtSvc)

	req := httptest.NewRequest(http.MethodGet, "/protected/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	var resp response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == nil || resp.Error.Message != "invalid or expired token" {
		t.Errorf("expected invalid or expired token message, got %v", resp.Error)
	}
}

func TestAuth_ExpiredToken(t *testing.T) {
	// Create JWT service with negative duration to generate expired tokens
	expiredSvc, err := infraservice.NewJWTService("super-secret-test-key-32bytes-long!", -1*time.Minute, -1*time.Minute)
	if err != nil {
		t.Fatalf("failed to create expired JWT service: %v", err)
	}

	validSvc := setupTestJWTService(t)
	engine := setupAuthTestEngine(validSvc)

	token, _, err := expiredSvc.GenerateAccessToken(domainservice.AccessTokenClaims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Roles:    []string{entity.RoleEmployee},
		Email:    "expired@example.com",
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	var resp response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == nil || resp.Error.Message != "token has expired" {
		t.Errorf("expected 'token has expired' message, got %v", resp.Error)
	}
}

func TestAuth_RefreshTokenRejectedAsAccessToken(t *testing.T) {
	jwtSvc := setupTestJWTService(t)
	engine := setupAuthTestEngine(jwtSvc)

	refreshToken, _, err := jwtSvc.GenerateRefreshToken(domainservice.RefreshTokenClaims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		TokenID:  uuid.New(),
		Family:   uuid.New(),
		Roles:    []string{entity.RoleEmployee},
	})
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected/me", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 when using refresh token as access token, got %d", rec.Code)
	}
}

func TestAuth_ValidAccessToken_SetsContext(t *testing.T) {
	jwtSvc := setupTestJWTService(t)
	engine := setupAuthTestEngine(jwtSvc)

	userID := uuid.New()
	tenantID := uuid.New()
	roles := []string{entity.RoleAdmin}
	email := "admin@acme.com"

	token, _, err := jwtSvc.GenerateAccessToken(domainservice.AccessTokenClaims{
		UserID:   userID,
		TenantID: tenantID,
		Roles:    roles,
		Email:    email,
	})
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var data map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data["gin_user_id"] != userID.String() {
		t.Errorf("expected gin_user_id %q, got %v", userID.String(), data["gin_user_id"])
	}
	if data["ctx_user_id"] != userID.String() {
		t.Errorf("expected ctx_user_id %q, got %v", userID.String(), data["ctx_user_id"])
	}
	if data["ctx_tenant_id"] != tenantID.String() {
		t.Errorf("expected ctx_tenant_id %q, got %v", tenantID.String(), data["ctx_tenant_id"])
	}
	if data["claims_email"] != email {
		t.Errorf("expected claims_email %q, got %v", email, data["claims_email"])
	}
}

func TestRequireRole_AccessControl(t *testing.T) {
	jwtSvc := setupTestJWTService(t)
	engine := setupAuthTestEngine(jwtSvc)

	tenantID := uuid.New()

	tests := []struct {
		name           string
		userRoles      []string
		targetPath     string
		expectedStatus int
	}{
		{
			name:           "Admin accessing admin-only endpoint",
			userRoles:      []string{entity.RoleAdmin},
			targetPath:     "/protected/admin-only",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Super Admin accessing admin-only endpoint",
			userRoles:      []string{entity.RoleSuperAdmin},
			targetPath:     "/protected/admin-only",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Employee accessing admin-only endpoint is forbidden",
			userRoles:      []string{entity.RoleEmployee},
			targetPath:     "/protected/admin-only",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Super Admin accessing super-admin-only endpoint",
			userRoles:      []string{entity.RoleSuperAdmin},
			targetPath:     "/protected/super-admin-only",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Admin accessing super-admin-only endpoint is forbidden",
			userRoles:      []string{entity.RoleAdmin},
			targetPath:     "/protected/super-admin-only",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Employee accessing employee-only endpoint",
			userRoles:      []string{entity.RoleEmployee},
			targetPath:     "/protected/employee-only",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Admin accessing employee-only endpoint is forbidden",
			userRoles:      []string{entity.RoleAdmin},
			targetPath:     "/protected/employee-only",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, _, err := jwtSvc.GenerateAccessToken(domainservice.AccessTokenClaims{
				UserID:   uuid.New(),
				TenantID: tenantID,
				Roles:    tc.userRoles,
				Email:    "test@example.com",
			})
			if err != nil {
				t.Fatalf("failed to generate access token: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, tc.targetPath, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRequireRole_UnauthenticatedAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/unprotected-admin", RequireRole(entity.RoleAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/unprotected-admin", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized when no user context exists, got %d", rec.Code)
	}
}

func TestRequireRole_AuthenticatedNoRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/authenticated-no-roles", func(c *gin.Context) {
		// Simulate Auth having run for a user with zero assigned roles: a valid
		// identity is present in context, but the roles slice is empty.
		c.Set(ContextKeyUserID, uuid.New().String())
		c.Set(ContextKeyRoles, []string{})

		reqCtx := ctx.WithUserID(c.Request.Context(), uuid.New().String())
		reqCtx = ctx.WithRoles(reqCtx, []string{})
		c.Request = c.Request.WithContext(reqCtx)

		c.Next()
	}, RequireRole(entity.RoleAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/authenticated-no-roles", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for authenticated user with zero roles, got %d", rec.Code)
	}

	var resp response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == nil || resp.Error.Message != "insufficient permissions" {
		t.Errorf("expected 'insufficient permissions' message, got %v", resp.Error)
	}
}

func TestRoleHelpers(t *testing.T) {
	roles := []string{entity.RoleAdmin, entity.RoleEmployee}

	if !HasRole(roles, entity.RoleAdmin) {
		t.Errorf("expected HasRole to return true for admin")
	}
	if !HasRole(roles, entity.RoleEmployee) {
		t.Errorf("expected HasRole to return true for employee")
	}
	if HasRole(roles, entity.RoleSuperAdmin) {
		t.Errorf("expected HasRole to return false for super_admin")
	}

	if !HasAnyRole(roles, entity.RoleSuperAdmin, entity.RoleAdmin) {
		t.Errorf("expected HasAnyRole to return true when at least one matches")
	}
	if HasAnyRole(roles, entity.RoleSuperAdmin, "custom_role") {
		t.Errorf("expected HasAnyRole to return false when none match")
	}
}
