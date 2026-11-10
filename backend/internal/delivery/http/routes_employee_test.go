package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

type gateState struct{ complete map[uuid.UUID]bool }

func (g *gateState) Execute(_ context.Context, _, userID uuid.UUID) (bool, error) {
	return g.complete[userID], nil
}

type okMe struct{}

func (okMe) Execute(_ context.Context, tenantID, userID uuid.UUID) (*emptypes.Me, error) {
	return &emptypes.Me{User: &entity.User{ID: userID}, OrganizationName: "Acme", FirstLogin: true, ProfileRequired: true}, nil
}

type okMyProfile struct{}

func (okMyProfile) Execute(context.Context, uuid.UUID, uuid.UUID) (*emptypes.MyProfile, error) {
	return &emptypes.MyProfile{}, nil
}

type okList struct{}

func (okList) Execute(context.Context, uuid.UUID, emptypes.ListEmployeesQuery) (*emptypes.ListEmployeesResult, error) {
	return &emptypes.ListEmployeesResult{}, nil
}

type okContacts struct{}

func (okContacts) Execute(context.Context, uuid.UUID, uuid.UUID) ([]*entity.EmergencyContact, error) {
	return nil, nil
}

func employeeGateEngine(t *testing.T, gate *gateState) (*gin.Engine, domainservice.TokenService, *fakeHolidayUseCases) {
	t.Helper()
	f := &fakeHolidayUseCases{}
	engine, jwtSvc := holidayEngineWith(t, f, func(ctr *container.AppContainer) {
		ctr.Employee = &container.EmployeeContainer{
			ProfileGate: gate,
			Handler: handlers.NewEmployeeHandler(handlers.EmployeeUseCases{
				Me: okMe{}, MyProfile: okMyProfile{}, List: okList{}, ListContacts: okContacts{},
			}),
		}
	})
	return engine, jwtSvc, f
}

func tokenFor(t *testing.T, svc domainservice.TokenService, tenant, user uuid.UUID, roles ...string) string {
	t.Helper()
	tok, _, err := svc.GenerateAccessToken(domainservice.AccessTokenClaims{UserID: user, TenantID: tenant, Roles: roles, Email: "a@b.com"})
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + tok
}

func TestProfileGating_EmployeeEndpoints(t *testing.T) {
	tenant, emp, admin, super := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	gate := &gateState{complete: map[uuid.UUID]bool{}}
	engine, svc, _ := employeeGateEngine(t, gate)

	gated := []string{
		"/api/v1/holidays?year=2026",
		"/api/v1/holiday-categories",
		"/api/v1/holiday-calendar/settings",
	}
	exempt := []string{"/api/v1/me", "/api/v1/me/profile", "/api/v1/me/emergency-contacts"}

	empTok := tokenFor(t, svc, tenant, emp, "employee")
	for _, p := range gated {
		rec := do(engine, http.MethodGet, p, empTok, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, p)
		assert.Contains(t, rec.Body.String(), `"code":"profile_incomplete"`, p)
	}
	for _, p := range exempt {
		assert.Equal(t, http.StatusOK, do(engine, http.MethodGet, p, empTok, "").Code, "%s stays reachable while incomplete", p)
	}

	// Flips after the profile is completed.
	gate.complete[emp] = true
	for _, p := range gated {
		assert.Equal(t, http.StatusOK, do(engine, http.MethodGet, p, empTok, "").Code, p)
	}

	// Admins and super admins are never gated, even with an incomplete profile.
	for _, tc := range []struct {
		id   uuid.UUID
		role string
	}{{admin, "admin"}, {super, "super_admin"}} {
		tok := tokenFor(t, svc, tenant, tc.id, tc.role)
		for _, p := range gated {
			assert.Equal(t, http.StatusOK, do(engine, http.MethodGet, p, tok, "").Code, "%s %s", tc.role, p)
		}
	}
}

func TestEmployeeRoutes_AuthorizationMatrix(t *testing.T) {
	tenant := uuid.New()
	gate := &gateState{complete: map[uuid.UUID]bool{}}
	engine, svc, _ := employeeGateEngine(t, gate)
	id := uuid.NewString()

	for _, p := range []string{
		"/api/v1/employees", "/api/v1/employees/" + id, "/api/v1/employees/" + id + "/assignments",
		"/api/v1/employees/" + id + "/emergency-contacts",
	} {
		assert.Equal(t, http.StatusUnauthorized, do(engine, http.MethodGet, p, "", "").Code, p)
		assert.Equal(t, http.StatusForbidden, do(engine, http.MethodGet, p, tokenFor(t, svc, tenant, uuid.New(), "employee"), "").Code, "employee: %s", p)
	}
	assert.Equal(t, http.StatusOK, do(engine, http.MethodGet, "/api/v1/employees", tokenFor(t, svc, tenant, uuid.New(), "admin"), "").Code)
	assert.Equal(t, http.StatusOK, do(engine, http.MethodGet, "/api/v1/employees", tokenFor(t, svc, tenant, uuid.New(), "super_admin"), "").Code)

	// Mutations are admin-only too; the role endpoint is super_admin-only.
	for _, r := range []struct{ method, path string }{
		{http.MethodPut, "/api/v1/employees/" + id},
		{http.MethodPatch, "/api/v1/employees/" + id},
		{http.MethodPost, "/api/v1/employees/" + id + "/deactivate"},
		{http.MethodPost, "/api/v1/employees/" + id + "/activate"},
		{http.MethodPut, "/api/v1/employees/" + id + "/role"},
	} {
		assert.Equal(t, http.StatusForbidden, do(engine, r.method, r.path, tokenFor(t, svc, tenant, uuid.New(), "employee"), `{}`).Code, "employee %s", r.path)
	}
	assert.Equal(t, http.StatusForbidden, do(engine, http.MethodPut, "/api/v1/employees/"+id+"/role", tokenFor(t, svc, tenant, uuid.New(), "admin"), `{"role":"admin"}`).Code, "admin cannot change roles")

	// /me requires authentication but no particular role.
	assert.Equal(t, http.StatusUnauthorized, do(engine, http.MethodGet, "/api/v1/me", "", "").Code)
}
