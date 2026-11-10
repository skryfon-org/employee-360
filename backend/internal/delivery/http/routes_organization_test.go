package http

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	orgtypes "github.com/skryfon-org/employee-360/backend/internal/types/organization"
)

type orgGet struct{ gotTenant *uuid.UUID }

func (g orgGet) Execute(_ context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error) {
	*g.gotTenant = tenantID
	return entity.DefaultOrganizationSettings(tenantID), nil
}

type orgUpdate struct{ gotTenant *uuid.UUID }

func (u orgUpdate) Execute(_ context.Context, tenantID, _ uuid.UUID, in orgtypes.UpdateSettingsInput) (*entity.OrganizationSettings, error) {
	*u.gotTenant = tenantID
	s := entity.DefaultOrganizationSettings(tenantID)
	s.EmployeeCodePrefix = in.EmployeeCodePrefix
	return s, nil
}

func orgRoutesEngine(t *testing.T, got *uuid.UUID) (*gin.Engine, domainservice.TokenService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtSvc := setupTestTokenService(t)
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: []string{"*"}}}
	ctr := testContainer(t, cfg, nil, nil)
	ctr.Auth = &container.AuthContainer{TokenService: jwtSvc, IdentityVerifier: verifierOf(jwtSvc)}
	ctr.Organization = &container.OrganizationContainer{Handler: handlers.NewOrganizationHandler(orgGet{got}, orgUpdate{got})}
	return SetupRouter(cfg, zerolog.Nop(), ctr), jwtSvc
}

func TestOrganizationSettingsRoutes_AdminAndSuperAdminOnly_TenantFromToken(t *testing.T) {
	var got uuid.UUID
	engine, jwtSvc := orgRoutesEngine(t, &got)
	body := `{"employee_code_prefix":"ACME","employee_code_padding":4,"employee_code_start_number":1,"tenant_id":"` + uuid.NewString() + `"}`
	for _, r := range []struct{ method, body string }{{http.MethodGet, ""}, {http.MethodPut, body}} {
		path := "/api/v1/organization/settings"
		if rec := do(engine, r.method, path, "", r.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s no token: want 401, got %d", r.method, rec.Code)
		}
		tenant := uuid.New()
		if rec := do(engine, r.method, path, bearer(t, jwtSvc, tenant, "employee"), r.body); rec.Code != http.StatusForbidden {
			t.Errorf("%s employee: want 403, got %d", r.method, rec.Code)
		}
		for _, role := range []string{"admin", "super_admin"} {
			got = uuid.Nil
			rec := do(engine, r.method, path, bearer(t, jwtSvc, tenant, role), r.body)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s %s: want 200, got %d: %s", r.method, role, rec.Code, rec.Body.String())
			}
			if got != tenant {
				t.Errorf("%s %s: usecase tenant %s, want token tenant %s (body tenant_id must be ignored)", r.method, role, got, tenant)
			}
		}
	}
}

type offDaysUC struct{ gotTenant *uuid.UUID }

func (u offDaysUC) Execute(_ context.Context, tenantID uuid.UUID) ([]int, error) {
	*u.gotTenant = tenantID
	return []int{0, 6}, nil
}

func TestWeeklyOffDaysRoute_EmployeeAllowed_TenantFromToken(t *testing.T) {
	var got uuid.UUID
	engine, jwtSvc := orgRoutesEngine(t, &got)
	// re-wire with the weekly off days handler
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: []string{"*"}}}
	ctr := testContainer(t, cfg, nil, nil)
	ctr.Auth = &container.AuthContainer{TokenService: jwtSvc, IdentityVerifier: verifierOf(jwtSvc)}
	ctr.Organization = &container.OrganizationContainer{
		Handler:       handlers.NewOrganizationHandler(orgGet{&got}, orgUpdate{&got}),
		WeeklyOffDays: handlers.NewWeeklyOffDaysHandler(offDaysUC{&got}),
	}
	engine = SetupRouter(cfg, zerolog.Nop(), ctr)

	path := "/api/v1/organization/weekly-off-days"
	if rec := do(engine, http.MethodGet, path, "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", rec.Code)
	}
	tenant := uuid.New()
	for _, role := range []string{"employee", "admin", "super_admin"} {
		got = uuid.Nil
		rec := do(engine, http.MethodGet, path+"?tenant_id="+uuid.NewString(), bearer(t, jwtSvc, tenant, role), "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: want 200, got %d: %s", role, rec.Code, rec.Body.String())
		}
		if got != tenant {
			t.Errorf("%s: usecase tenant %s, want token tenant %s", role, got, tenant)
		}
		if !strings.Contains(rec.Body.String(), `"weekly_off_days":[0,6]`) || strings.Contains(rec.Body.String(), "employee_code") {
			t.Errorf("%s: unexpected body %s", role, rec.Body.String())
		}
	}
}
