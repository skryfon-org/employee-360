package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

type lvListTypes struct{}

func (lvListTypes) Execute(context.Context, uuid.UUID, bool) ([]*entity.LeaveType, error) {
	return []*entity.LeaveType{}, nil
}

type lvCreateType struct{}

func (lvCreateType) Execute(_ context.Context, t, _ uuid.UUID, in leavetypes.CreateLeaveTypeInput) (*entity.LeaveType, error) {
	return &entity.LeaveType{ID: uuid.New(), TenantID: t, Name: in.Name, Code: in.Code, IsActive: true}, nil
}

type lvUpdateType struct{}

func (lvUpdateType) Execute(_ context.Context, t, _, id uuid.UUID, _ leavetypes.UpdateLeaveTypeInput) (*entity.LeaveType, error) {
	return &entity.LeaveType{ID: id, TenantID: t}, nil
}

type lvApprovers struct{}

func (lvApprovers) Execute(context.Context, uuid.UUID, uuid.UUID) ([]*entity.LeaveApprover, error) {
	return []*entity.LeaveApprover{}, nil
}

type lvApply struct{}

func (lvApply) Execute(_ context.Context, t, u uuid.UUID, _ leavetypes.ApplyLeaveInput) (*entity.LeaveRequestDetail, error) {
	return &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{ID: uuid.New(), TenantID: t, UserID: u}}, nil
}

type lvListMine struct{}

func (lvListMine) Execute(context.Context, uuid.UUID, uuid.UUID, leavetypes.ListMyLeaveQuery) (*leavetypes.LeaveRequestPage, error) {
	return &leavetypes.LeaveRequestPage{Page: 1, PageSize: 20}, nil
}

type lvCancel struct{}

func (lvCancel) Execute(_ context.Context, t, u, id uuid.UUID) (*entity.LeaveRequestDetail, error) {
	return &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{ID: id, TenantID: t, UserID: u}}, nil
}

type lvListAll struct{}

func (lvListAll) Execute(context.Context, uuid.UUID, leavetypes.ListLeaveRequestsQuery) (*leavetypes.LeaveRequestPage, error) {
	return &leavetypes.LeaveRequestPage{Page: 1, PageSize: 20}, nil
}

type lvDecide struct{}

func (lvDecide) Execute(_ context.Context, t, _ uuid.UUID, _ bool, id uuid.UUID, _ string) (*entity.LeaveRequestDetail, error) {
	return &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{ID: id, TenantID: t}}, nil
}

func leaveContainer() *container.LeaveContainer {
	return &container.LeaveContainer{
		Handler: handlers.NewLeaveTypeHandler(handlers.LeaveTypeUseCases{
			List: lvListTypes{}, Create: lvCreateType{}, Update: lvUpdateType{},
		}),
		RequestHandler: handlers.NewLeaveRequestHandler(handlers.LeaveRequestUseCases{
			ListApprovers: lvApprovers{}, Apply: lvApply{}, ListMy: lvListMine{}, Cancel: lvCancel{},
			ListTeam: lvListMine{}, ListAll: lvListAll{}, Approve: lvDecide{}, Reject: lvDecide{},
		}),
	}
}

func leaveRoutesEngine(t *testing.T, lc *container.LeaveContainer) (*gin.Engine, domainservice.TokenService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtSvc := setupTestTokenService(t)
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: []string{"*"}}}
	ctr := testContainer(t, cfg, nil, nil)
	ctr.Auth = &container.AuthContainer{TokenService: jwtSvc, IdentityVerifier: verifierOf(jwtSvc)}
	ctr.Leave = lc
	return SetupRouter(cfg, zerolog.Nop(), ctr), jwtSvc
}

func TestLeaveRoutes_AllRegistered(t *testing.T) {
	engine, _ := leaveRoutesEngine(t, leaveContainer())
	registered := map[string]bool{}
	for _, r := range engine.Routes() {
		registered[r.Method+" "+r.Path] = true
	}
	for _, route := range []string{
		"GET /api/v1/leave-types",
		"POST /api/v1/leave-types",
		"PATCH /api/v1/leave-types/:id",
		"GET /api/v1/me/leave-approvers",
		"GET /api/v1/me/leave-requests",
		"POST /api/v1/me/leave-requests",
		"POST /api/v1/me/leave-requests/:id/cancel",
		"GET /api/v1/leave-requests/team",
		"GET /api/v1/leave-requests",
		"POST /api/v1/leave-requests/:id/approve",
		"POST /api/v1/leave-requests/:id/reject",
	} {
		assert.True(t, registered[route], "expected route %s to be registered", route)
	}
	// Leave types are deactivated, never deleted; no route mutates requests otherwise.
	for _, route := range []string{
		"DELETE /api/v1/leave-types/:id",
		"PUT /api/v1/leave-types/:id",
		"DELETE /api/v1/leave-requests/:id",
		"PATCH /api/v1/leave-requests/:id",
	} {
		assert.False(t, registered[route], "route %s must not exist", route)
	}
}

func TestLeaveRoutes_Unauthenticated401(t *testing.T) {
	engine, _ := leaveRoutesEngine(t, leaveContainer())
	id := uuid.NewString()
	for _, rt := range leaveRouteCases(id) {
		rec := do(engine, rt.method, rt.path, "", rt.body)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "%s %s", rt.method, rt.path)
	}
}

type leaveRouteCase struct {
	method, path, body string
	want               int // status for an allowed caller
	adminOnly          bool
}

func leaveRouteCases(id string) []leaveRouteCase {
	apply := `{"leave_type_id":"` + uuid.NewString() + `","start_date":"2026-11-01","end_date":"2026-11-01","approver_id":"` + uuid.NewString() + `"}`
	return []leaveRouteCase{
		{http.MethodGet, "/api/v1/leave-types", "", http.StatusOK, false},
		{http.MethodPost, "/api/v1/leave-types", `{"name":"Study","code":"STUDY"}`, http.StatusCreated, true},
		{http.MethodPatch, "/api/v1/leave-types/" + id, `{"is_active":false}`, http.StatusOK, true},
		{http.MethodGet, "/api/v1/me/leave-approvers", "", http.StatusOK, false},
		{http.MethodGet, "/api/v1/me/leave-requests", "", http.StatusOK, false},
		{http.MethodPost, "/api/v1/me/leave-requests", apply, http.StatusCreated, false},
		{http.MethodPost, "/api/v1/me/leave-requests/" + id + "/cancel", "", http.StatusOK, false},
		{http.MethodGet, "/api/v1/leave-requests/team", "", http.StatusOK, false},
		{http.MethodGet, "/api/v1/leave-requests", "", http.StatusOK, true},
		{http.MethodPost, "/api/v1/leave-requests/" + id + "/approve", `{"comment":"ok"}`, http.StatusOK, false},
		{http.MethodPost, "/api/v1/leave-requests/" + id + "/reject", `{"comment":"no"}`, http.StatusOK, false},
	}
}

func TestLeaveRoutes_RoleGating(t *testing.T) {
	engine, jwtSvc := leaveRoutesEngine(t, leaveContainer())
	tenant := uuid.New()
	id := uuid.NewString()
	empTok := bearer(t, jwtSvc, tenant, "employee")
	adminTok := bearer(t, jwtSvc, tenant, "admin")
	superTok := bearer(t, jwtSvc, tenant, "super_admin")
	guestTok := bearer(t, jwtSvc, tenant, "guest")

	for _, rt := range leaveRouteCases(id) {
		name := rt.method + " " + rt.path

		// admin and super_admin reach every handler.
		for label, tok := range map[string]string{"admin": adminTok, "super_admin": superTok} {
			rec := do(engine, rt.method, rt.path, tok, rt.body)
			assert.Equal(t, rt.want, rec.Code, "%s as %s: %s", name, label, rec.Body.String())
		}

		// employee: forbidden on admin-only routes, reaches the handler otherwise.
		rec := do(engine, rt.method, rt.path, empTok, rt.body)
		if rt.adminOnly {
			assert.Equal(t, http.StatusForbidden, rec.Code, "%s as employee", name)
		} else {
			assert.Equal(t, rt.want, rec.Code, "%s as employee: %s", name, rec.Body.String())
		}

		// A token with no recognised role never gets in.
		rec = do(engine, rt.method, rt.path, guestTok, rt.body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s as unknown role", name)
	}
}

func TestLeaveRoutes_StaticSegmentsBeatIDParam(t *testing.T) {
	engine, jwtSvc := leaveRoutesEngine(t, leaveContainer())
	empTok := bearer(t, jwtSvc, uuid.New(), "employee")
	// "team" is not parsed as a request id, and the admin list stays admin-only.
	assert.Equal(t, http.StatusOK, do(engine, http.MethodGet, "/api/v1/leave-requests/team", empTok, "").Code)
	assert.Equal(t, http.StatusForbidden, do(engine, http.MethodGet, "/api/v1/leave-requests", empTok, "").Code)
}

func TestLeaveRoutes_NotRegisteredWithoutBothHandlers(t *testing.T) {
	// Like projects, the leave block is skipped (no panic) unless both handlers exist.
	for name, lc := range map[string]*container.LeaveContainer{
		"nil container":           nil,
		"missing type handler":    {RequestHandler: leaveContainer().RequestHandler},
		"missing request handler": {Handler: leaveContainer().Handler},
	} {
		assert.NotPanics(t, func() {
			engine, _ := leaveRoutesEngine(t, lc)
			for _, r := range engine.Routes() {
				assert.NotContains(t, r.Path, "leave", "%s: %s %s", name, r.Method, r.Path)
			}
		}, name)
	}
}
