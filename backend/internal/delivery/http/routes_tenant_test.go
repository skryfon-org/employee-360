package http

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	tenanttypes "github.com/skryfon-org/employee-360/backend/internal/types/tenant"
)

type tGet struct{}
type tRename struct{}
type tListDomains struct{}
type tAddDomain struct{}
type tUpdDomain struct{}
type tDelDomain struct{}

func okTenant(id uuid.UUID) *entity.Tenant {
	return &entity.Tenant{ID: id, Name: "T", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func (tGet) Execute(_ context.Context, id uuid.UUID) (*tenanttypes.TenantDetail, error) {
	return &tenanttypes.TenantDetail{Tenant: okTenant(id)}, nil
}
func (tRename) Execute(_ context.Context, _, id uuid.UUID, _ tenanttypes.UpdateTenantRequest) (*entity.Tenant, error) {
	return okTenant(id), nil
}
func (tListDomains) Execute(context.Context, uuid.UUID) ([]*entity.TenantDomain, error) {
	return nil, nil
}
func (tAddDomain) Execute(_ context.Context, _, id uuid.UUID, r tenanttypes.AddDomainRequest) (*entity.TenantDomain, error) {
	return &entity.TenantDomain{ID: uuid.New(), TenantID: id, Domain: r.Domain}, nil
}
func (tUpdDomain) Execute(_ context.Context, _, id, dom uuid.UUID, r tenanttypes.UpdateDomainRequest) (*entity.TenantDomain, error) {
	return &entity.TenantDomain{ID: dom, TenantID: id, Domain: r.Domain}, nil
}
func (tDelDomain) Execute(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error { return nil }

func tenantRoutesEngine(t *testing.T) (*gin.Engine, domainservice.TokenService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtSvc := setupTestTokenService(t)
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: []string{"*"}}}
	ctr := testContainer(t, cfg, nil, nil)
	ctr.Auth = &container.AuthContainer{
		TokenService:     jwtSvc,
		IdentityVerifier: verifierOf(jwtSvc),
		TenantHandler: handlers.NewTenantHandler(handlers.TenantUseCases{
			Get: tGet{}, Rename: tRename{}, ListDomains: tListDomains{}, AddDomain: tAddDomain{},
			UpdateDomain: tUpdDomain{}, DelDomain: tDelDomain{},
		}),
	}
	return SetupRouter(cfg, zerolog.Nop(), ctr), jwtSvc
}

func TestTenantRoutes_SuperAdminOnly(t *testing.T) {
	engine, jwtSvc := tenantRoutesEngine(t)
	dom := uuid.New().String()
	routes := []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/tenant", ""},
		{http.MethodPatch, "/api/v1/tenant", `{"name":"B"}`},
		{http.MethodGet, "/api/v1/tenant/domains", ""},
		{http.MethodPost, "/api/v1/tenant/domains", `{"domain":"b.com"}`},
		{http.MethodPatch, "/api/v1/tenant/domains/" + dom, `{"domain":"c.com"}`},
		{http.MethodDelete, "/api/v1/tenant/domains/" + dom, ""},
	}
	tenant := uuid.New()
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			if rec := do(engine, r.method, r.path, "", r.body); rec.Code != http.StatusUnauthorized {
				t.Errorf("no token: want 401, got %d", rec.Code)
			}
			if rec := do(engine, r.method, r.path, "Bearer garbage", r.body); rec.Code != http.StatusUnauthorized {
				t.Errorf("bad token: want 401, got %d", rec.Code)
			}
			for _, role := range []string{"employee", "admin"} {
				if rec := do(engine, r.method, r.path, bearer(t, jwtSvc, tenant, role), r.body); rec.Code != http.StatusForbidden {
					t.Errorf("%s: want 403, got %d: %s", role, rec.Code, rec.Body.String())
				}
			}
			rec := do(engine, r.method, r.path, bearer(t, jwtSvc, tenant, "super_admin"), r.body)
			if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
				t.Errorf("super_admin: want 2xx, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestTenantRoutes_OnlyCurrentTenantRoutesRegistered(t *testing.T) {
	engine, jwtSvc := tenantRoutesEngine(t)
	registered := map[string]bool{}
	for _, r := range engine.Routes() {
		registered[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"GET /api/v1/tenant", "PATCH /api/v1/tenant",
		"GET /api/v1/tenant/domains", "POST /api/v1/tenant/domains",
		"PATCH /api/v1/tenant/domains/:domainId", "DELETE /api/v1/tenant/domains/:domainId",
	} {
		if !registered[want] {
			t.Errorf("route not registered: %s", want)
		}
	}

	// Cross-tenant management routes are gone: 404 even for a super_admin.
	id := uuid.New().String()
	tok := bearer(t, jwtSvc, uuid.New(), "super_admin")
	for _, r := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/tenants"},
		{http.MethodPost, "/api/v1/tenants"},
		{http.MethodGet, "/api/v1/tenants/" + id},
		{http.MethodPatch, "/api/v1/tenants/" + id},
		{http.MethodPost, "/api/v1/tenants/" + id + "/activate"},
		{http.MethodPost, "/api/v1/tenants/" + id + "/deactivate"},
		{http.MethodGet, "/api/v1/tenants/" + id + "/domains"},
		{http.MethodPost, "/api/v1/tenants/" + id + "/domains"},
		{http.MethodDelete, "/api/v1/tenants/" + id + "/domains/" + id},
		{http.MethodGet, "/api/v1/tenant/" + id},
		{http.MethodPut, "/api/v1/tenant"},
		{http.MethodPost, "/api/v1/tenant"},
		{http.MethodDelete, "/api/v1/tenant"},
	} {
		if rec := do(engine, r.method, r.path, tok, `{}`); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: want 404, got %d", r.method, r.path, rec.Code)
		}
	}
	for k := range registered {
		if strings.Contains(k, "/tenants") {
			t.Errorf("unexpected tenants route registered: %s", k)
		}
	}
}
