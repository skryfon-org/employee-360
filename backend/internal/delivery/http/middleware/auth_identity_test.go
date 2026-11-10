package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

// mintRaw signs a token with the underlying JWT service WITHOUT registering the
// identity in the fake verifier, so each test controls the "database" state.
func mintRaw(t *testing.T, svc domainservice.TokenService, claims domainservice.AccessTokenClaims) string {
	t.Helper()
	inner := svc.(*registeringTokenService).TokenService
	tok, _, err := inner.GenerateAccessToken(claims)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return tok
}

func TestAuth_IdentityVerification(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()

	cases := []struct {
		name       string
		identity   *fakeIdentity // nil => user unknown
		tokenTen   uuid.UUID
		verifyErr  error
		wantStatus int
		wantCode   string
	}{
		{"valid", &fakeIdentity{TenantID: tenantA, Roles: []string{entity.RoleAdmin}}, tenantA, nil, http.StatusOK, ""},
		{"unknown user", nil, tenantA, nil, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"user in another tenant", &fakeIdentity{TenantID: tenantB, Roles: []string{entity.RoleAdmin}}, tenantA, nil, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"inactive user", &fakeIdentity{TenantID: tenantA, UserInactive: true}, tenantA, nil, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"soft-deleted user", &fakeIdentity{TenantID: tenantA, UserDeleted: true}, tenantA, nil, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"deleted tenant", &fakeIdentity{TenantID: tenantA, TenantDeleted: true}, tenantA, nil, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"inactive tenant", &fakeIdentity{TenantID: tenantA, TenantOff: true}, tenantA, nil, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"missing tenant", &fakeIdentity{TenantID: tenantA, TenantMissing: true}, tenantA, nil, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"verifier error fails closed", &fakeIdentity{TenantID: tenantA}, tenantA, errors.New("db down"), http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := setupTestJWTService(t)
			v := verifierFor(svc)
			userID := uuid.New()
			if tc.identity != nil {
				v.set(userID, *tc.identity)
			}
			v.Err = tc.verifyErr
			engine := setupAuthTestEngine(svc)

			tok := mintRaw(t, svc, domainservice.AccessTokenClaims{UserID: userID, TenantID: tc.tokenTen, Roles: []string{entity.RoleEmployee}, Email: "x@y.com"})
			req := httptest.NewRequest(http.MethodGet, "/protected/me", nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantCode == "" {
				return
			}
			var resp response.Envelope
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatal(err)
			}
			if resp.Error == nil || resp.Error.Code != tc.wantCode {
				t.Fatalf("error = %+v, want code %s", resp.Error, tc.wantCode)
			}
			// Must not reveal which check failed or leak internals.
			low := strings.ToLower(rec.Body.String())
			for _, leak := range []string{"tenant", "inactive", "deleted", "not found", "db down"} {
				if strings.Contains(low, leak) {
					t.Errorf("response leaks %q: %s", leak, rec.Body.String())
				}
			}
		})
	}
}

// Roles must come from the verifier (database), not the token claims.
func TestAuth_UsesCurrentRolesNotTokenRoles(t *testing.T) {
	tenant := uuid.New()
	svc := setupTestJWTService(t)
	v := verifierFor(svc)
	userID := uuid.New()
	// Token claims admin, but the user was demoted to employee in the DB.
	v.set(userID, fakeIdentity{TenantID: tenant, Roles: []string{entity.RoleEmployee}})
	engine := setupAuthTestEngine(svc)
	tok := mintRaw(t, svc, domainservice.AccessTokenClaims{UserID: userID, TenantID: tenant, Roles: []string{entity.RoleAdmin}, Email: "a@b.com"})

	do := func(path string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		return rec.Code
	}
	if got := do("/protected/admin-only"); got != http.StatusForbidden {
		t.Errorf("admin-only with stale admin token = %d, want 403", got)
	}
	if got := do("/protected/employee-only"); got != http.StatusOK {
		t.Errorf("employee-only = %d, want 200", got)
	}
}

func TestAuth_NilVerifierPanicsAtConstruction(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for nil verifier")
		}
	}()
	Auth(setupTestJWTService(t), nil)
}

func TestTenant_BehindAuth_RejectsDeletedTenant(t *testing.T) {
	tenant := uuid.New()
	svc := setupTestJWTService(t)
	v := verifierFor(svc)
	userID := uuid.New()
	v.set(userID, fakeIdentity{TenantID: tenant, TenantDeleted: true})
	engine := setupTenantTestEngine(svc)
	tok := mintRaw(t, svc, domainservice.AccessTokenClaims{UserID: userID, TenantID: tenant, Roles: []string{entity.RoleAdmin}, Email: "a@b.com"})

	req := httptest.NewRequest(http.MethodGet, "/tenant/resources", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
