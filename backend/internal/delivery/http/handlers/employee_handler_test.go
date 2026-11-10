package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/ctx"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	empimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/employee"
)

// Function-backed fakes for every employee usecase interface.
type (
	fnList    func(context.Context, uuid.UUID, emptypes.ListEmployeesQuery) (*emptypes.ListEmployeesResult, error)
	fnGet     func(context.Context, uuid.UUID, uuid.UUID) (*emptypes.EmployeeDetail, error)
	fnUpdate  func(context.Context, emptypes.Actor, uuid.UUID, emptypes.UpdateEmployeeInput) (*emptypes.EmployeeDetail, error)
	fnActor   func(context.Context, emptypes.Actor, uuid.UUID) (*emptypes.EmployeeDetail, error)
	fnRole    func(context.Context, emptypes.Actor, uuid.UUID, string) (*emptypes.EmployeeDetail, error)
	fnAssigns func(context.Context, uuid.UUID, uuid.UUID) ([]*entity.UserAssignment, error)
	fnMe      func(context.Context, uuid.UUID, uuid.UUID) (*emptypes.Me, error)
	fnMyProf  func(context.Context, uuid.UUID, uuid.UUID) (*emptypes.MyProfile, error)
	fnMyUpd   func(context.Context, uuid.UUID, uuid.UUID, emptypes.UpdateMyProfileInput) (*emptypes.MyProfile, error)
	fnAck     func(context.Context, uuid.UUID, uuid.UUID) error
	fnCCreate func(context.Context, emptypes.Actor, uuid.UUID, emptypes.EmergencyContactInput) (*entity.EmergencyContact, error)
)

func (f fnList) Execute(c context.Context, t uuid.UUID, q emptypes.ListEmployeesQuery) (*emptypes.ListEmployeesResult, error) {
	return f(c, t, q)
}
func (f fnGet) Execute(c context.Context, t, u uuid.UUID) (*emptypes.EmployeeDetail, error) {
	return f(c, t, u)
}
func (f fnUpdate) Execute(c context.Context, a emptypes.Actor, u uuid.UUID, in emptypes.UpdateEmployeeInput) (*emptypes.EmployeeDetail, error) {
	return f(c, a, u, in)
}
func (f fnActor) Execute(c context.Context, a emptypes.Actor, u uuid.UUID) (*emptypes.EmployeeDetail, error) {
	return f(c, a, u)
}
func (f fnRole) Execute(c context.Context, a emptypes.Actor, u uuid.UUID, r string) (*emptypes.EmployeeDetail, error) {
	return f(c, a, u, r)
}
func (f fnAssigns) Execute(c context.Context, t, u uuid.UUID) ([]*entity.UserAssignment, error) {
	return f(c, t, u)
}
func (f fnMe) Execute(c context.Context, t, u uuid.UUID) (*emptypes.Me, error) { return f(c, t, u) }
func (f fnMyProf) Execute(c context.Context, t, u uuid.UUID) (*emptypes.MyProfile, error) {
	return f(c, t, u)
}
func (f fnMyUpd) Execute(c context.Context, t, u uuid.UUID, in emptypes.UpdateMyProfileInput) (*emptypes.MyProfile, error) {
	return f(c, t, u, in)
}
func (f fnAck) Execute(c context.Context, t, u uuid.UUID) error { return f(c, t, u) }
func (f fnCCreate) Execute(c context.Context, a emptypes.Actor, u uuid.UUID, in emptypes.EmergencyContactInput) (*entity.EmergencyContact, error) {
	return f(c, a, u, in)
}

func empDetail(id uuid.UUID) *emptypes.EmployeeDetail {
	return &emptypes.EmployeeDetail{Summary: &entity.EmployeeSummary{UserID: id, Email: "e@x.test", Roles: []string{"employee"}}}
}

func empRouter(h *EmployeeHandler, tenant, user uuid.UUID, roles ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		rc := ctx.WithTenantID(c.Request.Context(), tenant.String())
		rc = ctx.WithUserID(rc, user.String())
		rc = ctx.WithRoles(rc, roles)
		c.Request = c.Request.WithContext(rc)
	})
	g := r.Group("/api/v1/employees")
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.PATCH("/:id", h.Update)
	g.GET("/:id/assignments", h.ListAssignments)
	g.POST("/:id/deactivate", h.Deactivate)
	g.POST("/:id/activate", h.Activate)
	g.PUT("/:id/role", h.ChangeRole)
	g.POST("/:id/emergency-contacts", h.CreateContact)
	me := r.Group("/api/v1/me")
	me.GET("", h.Me)
	me.GET("/profile", h.GetMyProfile)
	me.PATCH("/profile", h.UpdateMyProfile)
	me.POST("/emergency-contacts", h.CreateMyContact)
	return r
}

func call(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestEmployeeHandler_List_UsesContextTenantAndParsesFilters(t *testing.T) {
	tenant, user, dept := uuid.New(), uuid.New(), uuid.New()
	var gotTenant uuid.UUID
	var gotQ emptypes.ListEmployeesQuery
	h := NewEmployeeHandler(EmployeeUseCases{List: fnList(func(_ context.Context, tn uuid.UUID, q emptypes.ListEmployeesQuery) (*emptypes.ListEmployeesResult, error) {
		gotTenant, gotQ = tn, q
		return &emptypes.ListEmployeesResult{Employees: []*entity.EmployeeSummary{{UserID: uuid.New(), Email: "a@b.c"}}, Total: 41, Page: 2, PageSize: 20}, nil
	})})
	r := empRouter(h, tenant, user, "admin")

	rec := call(r, "GET", "/api/v1/employees?tenant_id="+uuid.NewString()+"&search=%20ann%20&status=inactive&department_id="+dept.String()+"&employment_status=probation&page=2", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, tenant, gotTenant, "client-supplied tenant_id is ignored")
	assert.Equal(t, "ann", gotQ.Filter.Search)
	require.NotNil(t, gotQ.Filter.IsActive)
	assert.False(t, *gotQ.Filter.IsActive)
	assert.Equal(t, dept, *gotQ.Filter.DepartmentID)
	assert.Equal(t, "probation", gotQ.Filter.EmploymentStatus)
	assert.Equal(t, 2, gotQ.Page)
	var env struct {
		Meta struct {
			TotalItems int64 `json:"total_items"`
			TotalPages int   `json:"total_pages"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.EqualValues(t, 41, env.Meta.TotalItems)
	assert.Equal(t, 3, env.Meta.TotalPages)

	for bad, msg := range map[string]string{
		"status=bogus":       "status must be active or inactive",
		"department_id=nope": "department_id must be a UUID",
		"position_id=nope":   "position_id must be a UUID",
		"page=x":             "page must be an integer",
	} {
		rec := call(r, "GET", "/api/v1/employees?"+bad, "")
		assert.Equal(t, http.StatusBadRequest, rec.Code, bad)
		assert.Contains(t, rec.Body.String(), msg, bad)
	}
}

func TestEmployeeHandler_ErrorMapping(t *testing.T) {
	tenant, user, target := uuid.New(), uuid.New(), uuid.New()
	cases := []struct {
		err  error
		want int
	}{
		{domainerrors.ErrUserNotFound, http.StatusNotFound},
		{domainerrors.ErrEmployeeCodeTaken, http.StatusConflict},
		{domainerrors.ErrLastActiveAdmin, http.StatusConflict},
		{domainerrors.ErrCannotActOnSelf, http.StatusForbidden},
		{domainerrors.ErrCannotActOnSuperAdmin, http.StatusForbidden},
		{domainerrors.ErrDepartmentNotFound, http.StatusBadRequest},
		{domainerrors.ErrDepartmentInactive, http.StatusConflict},
		{domainerrors.ErrAssignmentDateOrder, http.StatusBadRequest},
		{empimpl.ErrGenderInvalid, http.StatusBadRequest},
		{context.DeadlineExceeded, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		h := NewEmployeeHandler(EmployeeUseCases{
			Get: fnGet(func(context.Context, uuid.UUID, uuid.UUID) (*emptypes.EmployeeDetail, error) { return nil, tc.err }),
		})
		rec := call(empRouter(h, tenant, user, "admin"), "GET", "/api/v1/employees/"+target.String(), "")
		assert.Equal(t, tc.want, rec.Code, tc.err.Error())
	}
	h := NewEmployeeHandler(EmployeeUseCases{})
	assert.Equal(t, http.StatusBadRequest, call(empRouter(h, tenant, user, "admin"), "GET", "/api/v1/employees/not-a-uuid", "").Code)
}

func TestEmployeeHandler_Update_PassesActorAndOptionalSemantics(t *testing.T) {
	tenant, admin, target := uuid.New(), uuid.New(), uuid.New()
	var gotActor emptypes.Actor
	var gotIn emptypes.UpdateEmployeeInput
	h := NewEmployeeHandler(EmployeeUseCases{Update: fnUpdate(func(_ context.Context, a emptypes.Actor, u uuid.UUID, in emptypes.UpdateEmployeeInput) (*emptypes.EmployeeDetail, error) {
		gotActor, gotIn = a, in
		return empDetail(u), nil
	})})
	r := empRouter(h, tenant, admin, "admin")

	for _, method := range []string{"PUT", "PATCH"} {
		gotIn = emptypes.UpdateEmployeeInput{}
		rec := call(r, method, "/api/v1/employees/"+target.String(), `{"tenant_id":"`+uuid.NewString()+`","first_name":"Ann","manager_id":null,"joining_date":"2024-03-05","phone":"+91 99999 99999"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, tenant, gotActor.TenantID)
		assert.Equal(t, admin, gotActor.UserID)
		assert.Equal(t, []string{"admin"}, gotActor.Roles)
		assert.True(t, gotIn.FirstName.Set)
		assert.Equal(t, "Ann", gotIn.FirstName.Value)
		assert.True(t, gotIn.ManagerID.Set && gotIn.ManagerID.Null, "explicit null is distinguishable")
		assert.False(t, gotIn.DepartmentID.Set, "absent stays unset")
		assert.Equal(t, time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC), gotIn.JoiningDate.Value.Time)
		assert.Equal(t, "+91 99999 99999", gotIn.Phone.Value, "embedded self fields bind")
	}
	assert.Equal(t, http.StatusBadRequest, call(r, "PUT", "/api/v1/employees/"+target.String(), `{"joining_date":"05/03/2024"}`).Code)
	assert.Equal(t, http.StatusBadRequest, call(r, "PUT", "/api/v1/employees/"+target.String(), `{`).Code)
}

func TestEmployeeHandler_AccountActionsAndRole(t *testing.T) {
	tenant, user, target := uuid.New(), uuid.New(), uuid.New()
	var deact, act, role bool
	var gotRole string
	h := NewEmployeeHandler(EmployeeUseCases{
		Deactivate: fnActor(func(_ context.Context, a emptypes.Actor, u uuid.UUID) (*emptypes.EmployeeDetail, error) {
			deact = true
			assert.Equal(t, target, u)
			return empDetail(u), nil
		}),
		Activate: fnActor(func(_ context.Context, _ emptypes.Actor, u uuid.UUID) (*emptypes.EmployeeDetail, error) {
			act = true
			return empDetail(u), nil
		}),
		ChangeRole: fnRole(func(_ context.Context, _ emptypes.Actor, u uuid.UUID, r string) (*emptypes.EmployeeDetail, error) {
			role, gotRole = true, r
			return empDetail(u), nil
		}),
		Assignments: fnAssigns(func(_ context.Context, tn, u uuid.UUID) ([]*entity.UserAssignment, error) {
			assert.Equal(t, tenant, tn)
			return []*entity.UserAssignment{{ID: uuid.New(), EffectiveFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}, nil
		}),
	})
	r := empRouter(h, tenant, user, "super_admin")
	base := "/api/v1/employees/" + target.String()
	assert.Equal(t, 200, call(r, "POST", base+"/deactivate", "").Code)
	assert.Equal(t, 200, call(r, "POST", base+"/activate", "").Code)
	assert.Equal(t, 200, call(r, "PUT", base+"/role", `{"role":"admin"}`).Code)
	assert.True(t, deact && act && role)
	assert.Equal(t, "admin", gotRole)
	assert.Equal(t, 400, call(r, "PUT", base+"/role", `{"role":"super_admin"}`).Code, "super_admin can never be granted")
	assert.Equal(t, 400, call(r, "PUT", base+"/role", `{}`).Code)

	rec := call(r, "GET", base+"/assignments", "")
	require.Equal(t, 200, rec.Code)
	assert.Contains(t, rec.Body.String(), `"effective_from":"2024-01-01"`)
	assert.Contains(t, rec.Body.String(), `"is_current":true`)
}

func TestEmployeeHandler_Me_UsesAuthContextIdentityOnly(t *testing.T) {
	tenant, user := uuid.New(), uuid.New()
	var seenTenant, seenUser uuid.UUID
	h := NewEmployeeHandler(EmployeeUseCases{
		Me: fnMe(func(_ context.Context, tn, u uuid.UUID) (*emptypes.Me, error) {
			seenTenant, seenUser = tn, u
			return &emptypes.Me{User: &entity.User{ID: u, Email: "e@x.test"}, Roles: []string{"employee"}, OrganizationID: tn, OrganizationName: "Acme", FirstLogin: true, ProfileRequired: true}, nil
		}),
		MyProfile: fnMyProf(func(_ context.Context, tn, u uuid.UUID) (*emptypes.MyProfile, error) {
			return &emptypes.MyProfile{MissingFields: []string{"phone"}}, nil
		}),
		UpdateMyProf: fnMyUpd(func(_ context.Context, tn, u uuid.UUID, in emptypes.UpdateMyProfileInput) (*emptypes.MyProfile, error) {
			seenUser = u
			assert.Equal(t, "Kochi", in.City.Value)
			return &emptypes.MyProfile{Completed: true}, nil
		}),
		CreateContact: fnCCreate(func(_ context.Context, a emptypes.Actor, u uuid.UUID, in emptypes.EmergencyContactInput) (*entity.EmergencyContact, error) {
			assert.Equal(t, a.UserID, u, "own contacts target the caller")
			return &entity.EmergencyContact{ID: uuid.New(), Name: in.Name}, nil
		}),
	})
	r := empRouter(h, tenant, user, "employee")

	rec := call(r, "GET", "/api/v1/me?user_id="+uuid.NewString()+"&tenant_id="+uuid.NewString(), "")
	require.Equal(t, 200, rec.Code)
	assert.Equal(t, tenant, seenTenant)
	assert.Equal(t, user, seenUser)
	body := rec.Body.String()
	assert.Contains(t, body, `"organization_name":"Acme"`)
	assert.Contains(t, body, `"first_login":true`)
	assert.Contains(t, body, `"profile_completed":false`)

	assert.Equal(t, 200, call(r, "GET", "/api/v1/me/profile", "").Code)
	rec = call(r, "PATCH", "/api/v1/me/profile", `{"city":"Kochi"}`)
	assert.Equal(t, 200, rec.Code, rec.Body.String())
	assert.Equal(t, 201, call(r, "POST", "/api/v1/me/emergency-contacts", `{"name":"A","relationship":"mother","phone":"12345"}`).Code)
	assert.Equal(t, 400, call(r, "POST", "/api/v1/me/emergency-contacts", `{"name":"A"}`).Code, "required fields enforced")
}

func TestEmployeeHandler_Unauthenticated401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewEmployeeHandler(EmployeeUseCases{})
	r := gin.New()
	r.GET("/api/v1/me", h.Me)
	r.GET("/api/v1/employees", h.List)
	assert.Equal(t, 401, call(r, "GET", "/api/v1/me", "").Code)
	assert.Equal(t, 401, call(r, "GET", "/api/v1/employees", "").Code)
}
