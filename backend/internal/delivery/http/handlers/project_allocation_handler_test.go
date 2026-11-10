package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
)

type allocMock struct {
	listAllocs  func(tenantID, projectID uuid.UUID, scope entity.AllocationScope) ([]*entity.AllocationWithUser, error)
	createAlloc func(tenantID, actorID uuid.UUID, in prjtypes.CreateAllocationInput) (*prjtypes.AllocationResult, error)
	updateAlloc func(tenantID, actorID uuid.UUID, in prjtypes.UpdateAllocationInput) (*prjtypes.AllocationResult, error)
	deleteAlloc func(tenantID, actorID, id uuid.UUID) error
	userAllocs  func(tenantID, userID uuid.UUID, scope entity.AllocationScope) (*prjtypes.UserAllocationsResult, error)
	myProjects  func(tenantID, userID uuid.UUID, scope entity.AllocationScope) (*prjtypes.UserAllocationsResult, error)
}

func (m *allocMock) useCases() ProjectAllocationUseCases {
	return ProjectAllocationUseCases{
		ListByProject: prjListAllocs(m.listAllocs),
		Create:        prjCreateAlloc(m.createAlloc),
		Update:        prjUpdateAlloc(m.updateAlloc),
		Delete:        prjDeleteAlloc(m.deleteAlloc),
		ByUser:        prjUserAllocs(m.userAllocs),
		MyProjects:    prjMyProjects(m.myProjects),
	}
}

type prjListAllocs func(tenantID, projectID uuid.UUID, scope entity.AllocationScope) ([]*entity.AllocationWithUser, error)

func (f prjListAllocs) Execute(_ context.Context, t, p uuid.UUID, s entity.AllocationScope) ([]*entity.AllocationWithUser, error) {
	return f(t, p, s)
}

type prjCreateAlloc func(tenantID, actorID uuid.UUID, in prjtypes.CreateAllocationInput) (*prjtypes.AllocationResult, error)

func (f prjCreateAlloc) Execute(_ context.Context, t, a uuid.UUID, in prjtypes.CreateAllocationInput) (*prjtypes.AllocationResult, error) {
	return f(t, a, in)
}

type prjUpdateAlloc func(tenantID, actorID uuid.UUID, in prjtypes.UpdateAllocationInput) (*prjtypes.AllocationResult, error)

func (f prjUpdateAlloc) Execute(_ context.Context, t, a uuid.UUID, in prjtypes.UpdateAllocationInput) (*prjtypes.AllocationResult, error) {
	return f(t, a, in)
}

type prjDeleteAlloc func(tenantID, actorID, id uuid.UUID) error

func (f prjDeleteAlloc) Execute(_ context.Context, t, a, id uuid.UUID) error { return f(t, a, id) }

type prjMyProjects func(tenantID, userID uuid.UUID, scope entity.AllocationScope) (*prjtypes.UserAllocationsResult, error)

func (f prjMyProjects) Execute(_ context.Context, t, u uuid.UUID, s entity.AllocationScope) (*prjtypes.UserAllocationsResult, error) {
	return f(t, u, s)
}

type prjUserAllocs func(tenantID, userID uuid.UUID, scope entity.AllocationScope) (*prjtypes.UserAllocationsResult, error)

func (f prjUserAllocs) Execute(_ context.Context, t, u uuid.UUID, s entity.AllocationScope) (*prjtypes.UserAllocationsResult, error) {
	return f(t, u, s)
}

func newAllocationRouter(m *allocMock) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewProjectAllocationHandler(m.useCases())
	r := gin.New()
	g := r.Group("/api/v1", withIdentity(testTenantID, testActorID))
	g.GET("/projects/:id/allocations", h.ListProjectAllocations)
	g.POST("/projects/:id/allocations", h.CreateAllocation)
	g.PATCH("/allocations/:id", h.UpdateAllocation)
	g.DELETE("/allocations/:id", h.DeleteAllocation)
	g.GET("/employees/:id/allocations", h.ListEmployeeAllocations)
	g.GET("/me/projects", h.ListMyProjects)
	return r
}

func TestProjectAllocationHandler_Allocations(t *testing.T) {
	projectID, userID := uuid.New(), uuid.New()
	alloc := &entity.ProjectAllocation{ID: uuid.New(), ProjectID: projectID, UserID: userID, AllocationPercent: 80,
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	m := &allocMock{
		listAllocs: func(tenantID, p uuid.UUID, scope entity.AllocationScope) ([]*entity.AllocationWithUser, error) {
			if p != projectID {
				return nil, domainerrors.ErrProjectNotFound
			}
			return []*entity.AllocationWithUser{{ProjectAllocation: *alloc, UserFirstName: "Ada", UserLastName: "Lovelace"}}, nil
		},
		createAlloc: func(tenantID, actorID uuid.UUID, in prjtypes.CreateAllocationInput) (*prjtypes.AllocationResult, error) {
			assert.Equal(t, projectID, in.ProjectID)
			assert.Equal(t, userID, in.UserID)
			switch in.AllocationPercent {
			case 150:
				return nil, ErrTestInvalid
			case 10:
				return nil, domainerrors.ErrAllocationOverlap
			case 20:
				return nil, domainerrors.ErrAllocationUserInactive
			case 30:
				return nil, domainerrors.ErrUserNotFound
			case 40:
				return nil, domainerrors.ErrProjectArchived
			}
			return &prjtypes.AllocationResult{Allocation: alloc, UserTotalPercent: 130, OverAllocated: true}, nil
		},
		updateAlloc: func(tenantID, actorID uuid.UUID, in prjtypes.UpdateAllocationInput) (*prjtypes.AllocationResult, error) {
			if in.ID != alloc.ID {
				return nil, domainerrors.ErrAllocationNotFound
			}
			return &prjtypes.AllocationResult{Allocation: alloc, UserTotalPercent: 80}, nil
		},
		deleteAlloc: func(tenantID, actorID, id uuid.UUID) error {
			if id != alloc.ID {
				return domainerrors.ErrAllocationNotFound
			}
			return nil
		},
	}
	r := newAllocationRouter(m)
	base := "/api/v1/projects/" + projectID.String() + "/allocations"
	body := func(pct string) string {
		return `{"user_id":"` + userID.String() + `","allocation_percent":` + pct + `,"start_date":"2026-01-01"}`
	}

	rec := doProject(r, http.MethodPost, base, body("80"))
	require.Equal(t, http.StatusCreated, rec.Code)
	var env struct {
		Data struct {
			OverAllocated bool `json:"over_allocated"`
			Total         int  `json:"user_total_percent"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.True(t, env.Data.OverAllocated, "over-allocation succeeds and is flagged")
	assert.Equal(t, 130, env.Data.Total)

	assert.Equal(t, http.StatusBadRequest, doProject(r, http.MethodPost, base, body("150")).Code)
	assert.Equal(t, http.StatusConflict, doProject(r, http.MethodPost, base, body("10")).Code)
	assert.Equal(t, http.StatusConflict, doProject(r, http.MethodPost, base, body("20")).Code)
	assert.Equal(t, http.StatusNotFound, doProject(r, http.MethodPost, base, body("30")).Code)
	assert.Equal(t, http.StatusConflict, doProject(r, http.MethodPost, base, body("40")).Code, "archived project")
	assert.Equal(t, http.StatusBadRequest, doProject(r, http.MethodPost, base, `{"user_id":"nope"}`).Code)

	rec = doProject(r, http.MethodGet, base, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var listEnv struct {
		Data []struct {
			UserName string `json:"user_name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listEnv))
	require.Len(t, listEnv.Data, 1)
	assert.Equal(t, "Ada Lovelace", listEnv.Data[0].UserName)
	assert.Equal(t, http.StatusNotFound, doProject(r, http.MethodGet, "/api/v1/projects/"+uuid.NewString()+"/allocations", "").Code)

	// state query param: default all, valid values forwarded, invalid -> 400.
	var gotScope entity.AllocationScope
	m.listAllocs = func(_, p uuid.UUID, scope entity.AllocationScope) ([]*entity.AllocationWithUser, error) {
		gotScope = scope
		return []*entity.AllocationWithUser{{ProjectAllocation: *alloc, UserFirstName: "Ada", UserLastName: "Lovelace"}}, nil
	}
	r = newAllocationRouter(m)
	assert.Equal(t, http.StatusOK, doProject(r, http.MethodGet, base, "").Code)
	assert.Equal(t, entity.AllocationScopeAll, gotScope)
	for _, st := range []string{"all", "active", "upcoming", "past"} {
		assert.Equal(t, http.StatusOK, doProject(r, http.MethodGet, base+"?state="+st, "").Code)
		assert.Equal(t, entity.AllocationScope(st), gotScope)
	}
	gotScope = ""
	assert.Equal(t, http.StatusBadRequest, doProject(r, http.MethodGet, base+"?state=bogus", "").Code)
	assert.Empty(t, gotScope, "usecase not called for invalid state")

	one := "/api/v1/allocations/" + alloc.ID.String()
	assert.Equal(t, http.StatusOK, doProject(r, http.MethodPatch, one, `{"end_date":"2026-12-31"}`).Code)
	assert.Equal(t, http.StatusNotFound, doProject(r, http.MethodPatch, "/api/v1/allocations/"+uuid.NewString(), `{}`).Code)
	assert.Equal(t, http.StatusOK, doProject(r, http.MethodDelete, one, "").Code)
	assert.Equal(t, http.StatusNotFound, doProject(r, http.MethodDelete, "/api/v1/allocations/"+uuid.NewString(), "").Code)
}

func TestProjectAllocationHandler_UserViews(t *testing.T) {
	empID := uuid.New()
	m := &allocMock{userAllocs: func(tenantID, userID uuid.UUID, _ entity.AllocationScope) (*prjtypes.UserAllocationsResult, error) {
		assert.Equal(t, testTenantID, tenantID)
		if userID != empID && userID != testActorID {
			return nil, domainerrors.ErrUserNotFound
		}
		return &prjtypes.UserAllocationsResult{
			Allocations: []*entity.AllocationWithProject{{
				ProjectAllocation: entity.ProjectAllocation{ID: uuid.New(), AllocationPercent: 120 - 20, StartDate: time.Now()},
				ProjectName:       "Apollo", ProjectCode: "APO", ProjectStatus: "active",
			}},
			TotalPercent: 110, OverAllocated: true,
		}, nil
	}}
	r := newAllocationRouter(m)

	rec := doProject(r, http.MethodGet, "/api/v1/employees/"+empID.String()+"/allocations", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"over_allocated":true`)
	assert.Contains(t, rec.Body.String(), `"total_percent":110`)
	assert.Contains(t, rec.Body.String(), `"counts_toward_total":true`)
	assert.Equal(t, http.StatusNotFound, doProject(r, http.MethodGet, "/api/v1/employees/"+uuid.NewString()+"/allocations", "").Code)

	// scope query param on the admin employee view.
	var empScope entity.AllocationScope = "unset"
	inner := m.userAllocs
	m.userAllocs = func(tn, u uuid.UUID, sc entity.AllocationScope) (*prjtypes.UserAllocationsResult, error) {
		empScope = sc
		return inner(tn, u, sc)
	}
	r = newAllocationRouter(m)
	empURL := "/api/v1/employees/" + empID.String() + "/allocations"
	assert.Equal(t, http.StatusOK, doProject(r, http.MethodGet, empURL, "").Code)
	assert.Equal(t, entity.AllocationScopeActive, empScope, "default scope is active")
	for _, sc := range []string{"active", "upcoming", "past", "all"} {
		assert.Equal(t, http.StatusOK, doProject(r, http.MethodGet, empURL+"?scope="+sc, "").Code)
		assert.Equal(t, entity.AllocationScope(sc), empScope)
	}
	empScope = "unset"
	assert.Equal(t, http.StatusBadRequest, doProject(r, http.MethodGet, empURL+"?scope=bogus", "").Code)
	assert.Equal(t, entity.AllocationScope("unset"), empScope, "usecase not called on invalid scope")

	// /me/projects uses the caller's id from the auth context, never a client value,
	// and its own (live-projects-only) usecase.
	var gotScope entity.AllocationScope
	m.myProjects = func(tenantID, userID uuid.UUID, scope entity.AllocationScope) (*prjtypes.UserAllocationsResult, error) {
		assert.Equal(t, testActorID, userID)
		gotScope = scope
		return &prjtypes.UserAllocationsResult{TotalPercent: 40}, nil
	}
	r = newAllocationRouter(m)
	rec = doProject(r, http.MethodGet, "/api/v1/me/projects", "")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, entity.AllocationScopeActive, gotScope, "default scope is active")
	for _, sc := range []string{"active", "upcoming", "past", "all"} {
		rec = doProject(r, http.MethodGet, "/api/v1/me/projects?scope="+sc, "")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, entity.AllocationScope(sc), gotScope)
	}
	gotScope = ""
	rec = doProject(r, http.MethodGet, "/api/v1/me/projects?scope=bogus", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, gotScope, "usecase not called on invalid scope")
}

func TestProjectAllocationHandler_AuthContextAndErrorMapping(t *testing.T) {
	called := false
	m := &allocMock{
		createAlloc: func(tenantID, actorID uuid.UUID, in prjtypes.CreateAllocationInput) (*prjtypes.AllocationResult, error) {
			called = true
			return nil, domainerrors.ErrForbidden
		},
		myProjects: func(tenantID, userID uuid.UUID, _ entity.AllocationScope) (*prjtypes.UserAllocationsResult, error) {
			return nil, assert.AnError
		},
		deleteAlloc: func(tenantID, actorID, id uuid.UUID) error { return domainerrors.ErrUnauthorized },
	}
	gin.SetMode(gin.TestMode)
	h := NewProjectAllocationHandler(m.useCases())
	bare := gin.New() // no identity middleware
	bare.POST("/projects/:id/allocations", h.CreateAllocation)
	bare.GET("/me/projects", h.ListMyProjects)
	bare.DELETE("/allocations/:id", h.DeleteAllocation)
	bare.GET("/employees/:id/allocations", h.ListEmployeeAllocations)
	id := uuid.NewString()
	body := `{"user_id":"` + uuid.NewString() + `","allocation_percent":10,"start_date":"2026-01-01"}`
	assert.Equal(t, http.StatusUnauthorized, doProject(bare, http.MethodPost, "/projects/"+id+"/allocations", body).Code)
	assert.Equal(t, http.StatusUnauthorized, doProject(bare, http.MethodGet, "/me/projects", "").Code)
	assert.Equal(t, http.StatusUnauthorized, doProject(bare, http.MethodDelete, "/allocations/"+id, "").Code)
	assert.Equal(t, http.StatusUnauthorized, doProject(bare, http.MethodGet, "/employees/"+id+"/allocations", "").Code)
	assert.False(t, called)

	r := newAllocationRouter(m)
	assert.Equal(t, http.StatusForbidden, doProject(r, http.MethodPost, "/api/v1/projects/"+id+"/allocations", body).Code)
	assert.Equal(t, http.StatusInternalServerError, doProject(r, http.MethodGet, "/api/v1/me/projects", "").Code)
	assert.Equal(t, http.StatusUnauthorized, doProject(r, http.MethodDelete, "/api/v1/allocations/"+id, "").Code)
}
