package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/ctx"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/shared"
)

func setupTenantTestEngine(tokenSvc domainservice.TokenService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	tenantGroup := engine.Group("/tenant", Auth(tokenSvc, verifierFor(tokenSvc)), Tenant())
	{
		tenantGroup.GET("/resources", func(c *gin.Context) {
			tenantID := c.GetString(ContextKeyTenantID)
			if tenantID == "" {
				response.Unauthorized(c, "failed to get tenant ID")
				return
			}

			ctxTenantID, _ := ctx.TenantIDFromContext(c.Request.Context())

			c.JSON(http.StatusOK, gin.H{
				"tenant_id":     tenantID,
				"ctx_tenant_id": ctxTenantID,
			})
		})

		tenantGroup.POST("/resources", func(c *gin.Context) {
			tenantID := c.GetString(ContextKeyTenantID)
			if tenantID == "" {
				response.Unauthorized(c, "failed to get tenant ID")
				return
			}

			ctxTenantID, _ := ctx.TenantIDFromContext(c.Request.Context())

			c.JSON(http.StatusCreated, gin.H{
				"tenant_id":     tenantID,
				"ctx_tenant_id": ctxTenantID,
			})
		})
	}

	// Route that tests Tenant() alone without Auth()
	engine.GET("/unauthenticated-tenant", Tenant(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return engine
}

func TestTenant_ValidContext(t *testing.T) {
	jwtSvc := setupTestJWTService(t)
	engine := setupTenantTestEngine(jwtSvc)

	expectedTenantID := uuid.New()
	token, _, err := jwtSvc.GenerateAccessToken(domainservice.AccessTokenClaims{
		UserID:   uuid.New(),
		TenantID: expectedTenantID,
		Roles:    []string{entity.RoleAdmin},
		Email:    "tenant-admin@acme.com",
	})
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tenant/resources", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var data map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if data["tenant_id"] != expectedTenantID.String() {
		t.Errorf("expected tenant_id %q, got %v", expectedTenantID.String(), data["tenant_id"])
	}
	if data["ctx_tenant_id"] != expectedTenantID.String() {
		t.Errorf("expected ctx_tenant_id %q, got %v", expectedTenantID.String(), data["ctx_tenant_id"])
	}
}

func TestTenant_StrictIsolation_IgnoresClientSuppliedTenantID(t *testing.T) {
	jwtSvc := setupTestJWTService(t)
	engine := setupTenantTestEngine(jwtSvc)

	legitimateTenantID := uuid.New()
	spoofedTenantID := uuid.New()

	token, _, err := jwtSvc.GenerateAccessToken(domainservice.AccessTokenClaims{
		UserID:   uuid.New(),
		TenantID: legitimateTenantID,
		Roles:    []string{entity.RoleEmployee},
		Email:    "emp@legit.com",
	})
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	// 1. Client attempts to spoof via X-Tenant-ID header and query param
	req := httptest.NewRequest(http.MethodGet, "/tenant/resources?tenant_id="+spoofedTenantID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(shared.TenantIDHeader, spoofedTenantID.String())
	req.Header.Set("X-Tenant-ID", spoofedTenantID.String())
	req.Header.Set("Tenant-ID", spoofedTenantID.String())
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var data map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Must match legitimateTenantID, NEVER spoofedTenantID
	if data["tenant_id"] != legitimateTenantID.String() {
		t.Fatalf("CRITICAL: tenant_id was not resolved from JWT! expected %s, got %v", legitimateTenantID.String(), data["tenant_id"])
	}
	if data["ctx_tenant_id"] != legitimateTenantID.String() {
		t.Fatalf("CRITICAL: ctx_tenant_id was not resolved from JWT! expected %s, got %v", legitimateTenantID.String(), data["ctx_tenant_id"])
	}

	// 2. Client attempts to spoof via POST JSON body payload
	bodyJSON := `{"tenant_id": "` + spoofedTenantID.String() + `", "name": "spoofed"}`
	reqPost := httptest.NewRequest(http.MethodPost, "/tenant/resources", strings.NewReader(bodyJSON))
	reqPost.Header.Set("Authorization", "Bearer "+token)
	reqPost.Header.Set("Content-Type", "application/json")
	reqPost.Header.Set(shared.TenantIDHeader, spoofedTenantID.String())
	recPost := httptest.NewRecorder()
	engine.ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recPost.Code, recPost.Body.String())
	}

	var postData map[string]interface{}
	if err := json.Unmarshal(recPost.Body.Bytes(), &postData); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if postData["tenant_id"] != legitimateTenantID.String() {
		t.Fatalf("CRITICAL: POST tenant_id trusted spoofed body! expected %s, got %v", legitimateTenantID.String(), postData["tenant_id"])
	}
	if postData["ctx_tenant_id"] != legitimateTenantID.String() {
		t.Fatalf("CRITICAL: POST ctx_tenant_id trusted spoofed body! expected %s, got %v", legitimateTenantID.String(), postData["ctx_tenant_id"])
	}
}

func TestTenant_MissingContext_Aborts(t *testing.T) {
	jwtSvc := setupTestJWTService(t)
	engine := setupTenantTestEngine(jwtSvc)

	req := httptest.NewRequest(http.MethodGet, "/unauthenticated-tenant", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 when tenant context is missing, got %d", rec.Code)
	}
}
