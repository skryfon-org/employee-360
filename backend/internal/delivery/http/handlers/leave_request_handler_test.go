package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/ctx"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

type applyFn func(ctx context.Context, tenantID, userID uuid.UUID, in leavetypes.ApplyLeaveInput) (*entity.LeaveRequestDetail, error)

func (f applyFn) Execute(c context.Context, t, u uuid.UUID, in leavetypes.ApplyLeaveInput) (*entity.LeaveRequestDetail, error) {
	return f(c, t, u, in)
}

type decideFn func(ctx context.Context, tenantID, actorID uuid.UUID, isAdmin bool, id uuid.UUID, comment string) (*entity.LeaveRequestDetail, error)

func (f decideFn) Execute(c context.Context, t, a uuid.UUID, isAdmin bool, id uuid.UUID, comment string) (*entity.LeaveRequestDetail, error) {
	return f(c, t, a, isAdmin, id, comment)
}

func withLeaveIdentity(tenantID, userID uuid.UUID, roles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rc := ctx.WithTenantID(c.Request.Context(), tenantID.String())
		rc = ctx.WithUserID(rc, userID.String())
		rc = ctx.WithRoles(rc, roles)
		c.Request = c.Request.WithContext(rc)
	}
}

func leaveRouter(uc LeaveRequestUseCases, tenantID, userID uuid.UUID, roles ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewLeaveRequestHandler(uc)
	r := gin.New()
	g := r.Group("/api/v1", withLeaveIdentity(tenantID, userID, roles))
	g.POST("/me/leave-requests", h.ApplyLeave)
	g.GET("/me/leave-requests", h.ListMyLeaveRequests)
	g.GET("/leave-requests/team", h.ListTeamLeaveRequests)
	g.POST("/leave-requests/:id/approve", h.ApproveLeaveRequest)
	g.POST("/leave-requests/:id/reject", h.RejectLeaveRequest)
	return r
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestLeaveHandler_ApplyErrorMapping(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{domainerrors.ErrLeaveOverlap, 422, "leave_overlap"},
		{domainerrors.ErrLeaveStartTooOld, 422, "leave_start_too_old"},
		{domainerrors.ErrInvalidLeaveApprover, 422, "invalid_approver"},
		{domainerrors.ErrLeaveTypeInactive, 422, "leave_type_inactive"},
		{domainerrors.ErrLeaveTypeNotFound, 404, ""},
		{domainerrors.ErrInvalidLeaveData, 400, ""},
	}
	body := `{"leave_type_id":"` + uuid.NewString() + `","start_date":"2026-11-01","end_date":"2026-11-01","approver_id":"` + uuid.NewString() + `"}`
	for _, tc := range cases {
		var gotTenant, gotUser uuid.UUID
		uc := LeaveRequestUseCases{Apply: applyFn(func(_ context.Context, t, u uuid.UUID, _ leavetypes.ApplyLeaveInput) (*entity.LeaveRequestDetail, error) {
			gotTenant, gotUser = t, u
			return nil, tc.err
		})}
		rec := do(leaveRouter(uc, tenantID, userID, entity.RoleEmployee), "POST", "/api/v1/me/leave-requests", body)
		assert.Equal(t, tc.status, rec.Code, tc.err)
		if tc.code != "" {
			assert.Contains(t, rec.Body.String(), tc.code)
		}
		assert.Equal(t, tenantID, gotTenant, "tenant from auth context")
		assert.Equal(t, userID, gotUser, "applicant from auth context")
	}

	// Malformed body never reaches the usecase.
	called := false
	uc := LeaveRequestUseCases{Apply: applyFn(func(context.Context, uuid.UUID, uuid.UUID, leavetypes.ApplyLeaveInput) (*entity.LeaveRequestDetail, error) {
		called = true
		return nil, nil
	})}
	rec := do(leaveRouter(uc, tenantID, userID, entity.RoleEmployee), "POST", "/api/v1/me/leave-requests", `{"start_date":"nope"}`)
	assert.Equal(t, 400, rec.Code)
	assert.False(t, called)
}

func TestLeaveHandler_ApplySuccessIgnoresClientTenant(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	var got leavetypes.ApplyLeaveInput
	var gotTenant uuid.UUID
	uc := LeaveRequestUseCases{Apply: applyFn(func(_ context.Context, t, _ uuid.UUID, in leavetypes.ApplyLeaveInput) (*entity.LeaveRequestDetail, error) {
		gotTenant, got = t, in
		return &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{ID: uuid.New(), Status: "pending"}}, nil
	})}
	body := `{"tenant_id":"` + uuid.NewString() + `","leave_type_id":"` + uuid.NewString() + `","start_date":"2026-11-01","end_date":"2026-11-02","day_part":"full","reason":" r ","approver_id":"` + uuid.NewString() + `"}`
	rec := do(leaveRouter(uc, tenantID, userID, entity.RoleEmployee), "POST", "/api/v1/me/leave-requests", body)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	assert.Equal(t, tenantID, gotTenant)
	assert.Equal(t, "2026-11-02", got.EndDate.Format("2006-01-02"))
}

func TestLeaveHandler_DecisionMapping(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	id := uuid.NewString()
	cases := []struct {
		err    error
		status int
	}{
		{domainerrors.ErrLeaveSelfDecision, 403},
		{domainerrors.ErrLeaveRequestNotFound, 404},
		{domainerrors.ErrLeaveNotPending, 409},
		{nil, 200},
	}
	for _, tc := range cases {
		var gotAdmin bool
		var gotComment string
		dec := decideFn(func(_ context.Context, _, _ uuid.UUID, isAdmin bool, _ uuid.UUID, comment string) (*entity.LeaveRequestDetail, error) {
			gotAdmin, gotComment = isAdmin, comment
			if tc.err != nil {
				return nil, tc.err
			}
			return &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{ID: uuid.New()}}, nil
		})
		uc := LeaveRequestUseCases{Approve: dec, Reject: dec}
		r := leaveRouter(uc, tenantID, userID, entity.RoleEmployee)
		rec := do(r, "POST", "/api/v1/leave-requests/"+id+"/reject", `{"comment":"no"}`)
		assert.Equal(t, tc.status, rec.Code, tc.err)
		assert.False(t, gotAdmin, "employee role is not admin")
		if tc.err == nil {
			assert.Equal(t, "no", gotComment)
		}

		rec = do(leaveRouter(uc, tenantID, userID, entity.RoleAdmin), "POST", "/api/v1/leave-requests/"+id+"/approve", "")
		assert.Equal(t, tc.status, rec.Code, "empty body allowed on approve")
		assert.True(t, gotAdmin, "admin role detected")
	}
	rec := do(leaveRouter(LeaveRequestUseCases{}, tenantID, userID), "POST", "/api/v1/leave-requests/not-a-uuid/approve", "")
	assert.Equal(t, 400, rec.Code)
}

type listPageFn func(ctx context.Context, tenantID, userID uuid.UUID, q leavetypes.ListMyLeaveQuery) (*leavetypes.LeaveRequestPage, error)

func (f listPageFn) Execute(c context.Context, t, u uuid.UUID, q leavetypes.ListMyLeaveQuery) (*leavetypes.LeaveRequestPage, error) {
	return f(c, t, u, q)
}

func TestLeaveHandler_CallerListsArePaginated(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	var gotTenant, gotUser uuid.UUID
	var gotQ leavetypes.ListMyLeaveQuery
	fn := listPageFn(func(_ context.Context, tn, u uuid.UUID, q leavetypes.ListMyLeaveQuery) (*leavetypes.LeaveRequestPage, error) {
		gotTenant, gotUser, gotQ = tn, u, q
		return &leavetypes.LeaveRequestPage{Items: []*entity.LeaveRequestDetail{}, Total: 45, Page: q.Page, PageSize: q.PageSize}, nil
	})
	r := leaveRouter(LeaveRequestUseCases{ListMy: fn, ListTeam: fn}, tenantID, userID, entity.RoleEmployee)

	for _, path := range []string{"/api/v1/me/leave-requests", "/api/v1/leave-requests/team"} {
		rec := do(r, "GET", path+"?page=2&page_size=10&status=pending&from=2026-01-01&to=2026-01-31&leave_type_id=abc", "")
		require.Equal(t, 200, rec.Code, rec.Body.String())
		assert.Equal(t, tenantID, gotTenant, path)
		assert.Equal(t, userID, gotUser, path)
		assert.Equal(t, leavetypes.ListMyLeaveQuery{Page: 2, PageSize: 10, Status: "pending", From: "2026-01-01", To: "2026-01-31", LeaveTypeID: "abc"}, gotQ, path)
		assert.Contains(t, rec.Body.String(), `"total_items":45`, path)
		assert.Contains(t, rec.Body.String(), `"total_pages":5`, path)
		assert.Contains(t, rec.Body.String(), `"page":2`, path)

		if path == "/api/v1/leave-requests/team" {
			rec = do(r, "GET", path+"?exclude_status=pending", "")
			require.Equal(t, 200, rec.Code, rec.Body.String())
			assert.Equal(t, "pending", gotQ.ExcludeStatus)
			assert.Empty(t, gotQ.Status)
		}

		assert.Equal(t, 400, do(r, "GET", path+"?page=abc", "").Code, path)
		assert.Equal(t, 400, do(r, "GET", path+"?page_size=x", "").Code, path)
	}
}
