package http

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	dashtypes "github.com/skryfon-org/employee-360/backend/internal/types/dashboard"
)

type fakeEmployeeDash struct{ gotTenant, gotUser uuid.UUID }

func (f *fakeEmployeeDash) Execute(_ context.Context, tenantID, userID uuid.UUID) (*dashtypes.EmployeeDashboardResponse, error) {
	f.gotTenant, f.gotUser = tenantID, userID
	pa := 2
	return &dashtypes.EmployeeDashboardResponse{
		Greeting: dashtypes.EmployeeGreeting{FirstName: "Asha", Today: dashtypes.EmployeeToday{Date: "2026-10-30", Kind: "working_day"}},
		Stats:    dashtypes.EmployeeStats{PendingApprovals: &pa},
		Leave:    dashtypes.EmployeeLeaveOverview{Recent: []dashtypes.EmployeeLeaveItem{}},
		Holidays: []dashtypes.EmployeeHoliday{},
		Projects: []dashtypes.EmployeeProject{},
		Week:     dashtypes.EmployeeWeek{Days: []dashtypes.EmployeeWeekDay{}},
	}, nil
}

func employeeDashEngine(t *testing.T, uc *fakeEmployeeDash) (*gin.Engine, domainservice.TokenService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtSvc := setupTestTokenService(t)
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: []string{"*"}}}
	ctr := testContainer(t, cfg, nil, nil)
	ctr.Auth = &container.AuthContainer{
		TokenService:     jwtSvc,
		IdentityVerifier: verifierOf(jwtSvc),
		DashboardHandler: handlers.NewDashboardHandler(&fakeAdminDash{}, &fakeSuperDash{}).WithEmployee(uc),
	}
	ctr.Employee = &container.EmployeeContainer{ProfileGate: passGate{}}
	return SetupRouter(cfg, zerolog.Nop(), ctr), jwtSvc
}

func TestEmployeeDashboardRoute_Unauthenticated401(t *testing.T) {
	engine, _ := employeeDashEngine(t, &fakeEmployeeDash{})
	assert.Equal(t, http.StatusUnauthorized, do(engine, http.MethodGet, "/api/v1/dashboard/employee", "", "").Code)
}

func TestEmployeeDashboardRoute_WrongRole403(t *testing.T) {
	engine, jwtSvc := employeeDashEngine(t, &fakeEmployeeDash{})
	tok := bearer(t, jwtSvc, uuid.New(), "guest")
	assert.Equal(t, http.StatusForbidden, do(engine, http.MethodGet, "/api/v1/dashboard/employee", tok, "").Code)
}

func TestEmployeeDashboardRoute_OK200Shape(t *testing.T) {
	uc := &fakeEmployeeDash{}
	engine, jwtSvc := employeeDashEngine(t, uc)
	for _, role := range []string{"employee", "admin", "super_admin"} {
		tenant := uuid.New()
		rec := do(engine, http.MethodGet, "/api/v1/dashboard/employee?tenant_id="+uuid.NewString(), bearer(t, jwtSvc, tenant, role), "")
		require.Equal(t, http.StatusOK, rec.Code, role+": "+rec.Body.String())
		assert.Equal(t, tenant, uc.gotTenant, "tenant must come from the token")

		var body struct {
			Data map[string]json.RawMessage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		for _, k := range []string{"greeting", "stats", "leave", "holidays", "projects", "week"} {
			assert.Contains(t, body.Data, k, role)
		}
	}
}
