package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

type prjMock struct {
	create  func(tenantID, actorID uuid.UUID, in prjtypes.CreateProjectInput) (*entity.Project, error)
	get     func(tenantID, id uuid.UUID) (*entity.Project, error)
	list    func(tenantID uuid.UUID, q prjtypes.ListProjectsQuery) (*prjtypes.ListProjectsResult, error)
	update  func(tenantID, actorID uuid.UUID, in prjtypes.UpdateProjectInput) (*entity.Project, error)
	suggest func(tenantID uuid.UUID, name string) (string, error)
}

func (m *prjMock) useCases() ProjectUseCases {
	return ProjectUseCases{Create: prjCreate(m.create), Get: prjGet(m.get), List: prjList(m.list), Update: prjUpdate(m.update), SuggestCode: prjSuggest(m.suggest)}
}

type prjCreate func(tenantID, actorID uuid.UUID, in prjtypes.CreateProjectInput) (*entity.Project, error)

func (f prjCreate) Execute(_ context.Context, t, a uuid.UUID, in prjtypes.CreateProjectInput) (*entity.Project, error) {
	return f(t, a, in)
}

type prjSuggest func(tenantID uuid.UUID, name string) (string, error)

func (f prjSuggest) Execute(_ context.Context, t uuid.UUID, name string) (string, error) {
	return f(t, name)
}

type prjGet func(tenantID, id uuid.UUID) (*entity.Project, error)

func (f prjGet) Execute(_ context.Context, t, id uuid.UUID) (*entity.Project, error) { return f(t, id) }

type prjList func(tenantID uuid.UUID, q prjtypes.ListProjectsQuery) (*prjtypes.ListProjectsResult, error)

func (f prjList) Execute(_ context.Context, t uuid.UUID, q prjtypes.ListProjectsQuery) (*prjtypes.ListProjectsResult, error) {
	return f(t, q)
}

type prjUpdate func(tenantID, actorID uuid.UUID, in prjtypes.UpdateProjectInput) (*entity.Project, error)

func (f prjUpdate) Execute(_ context.Context, t, a uuid.UUID, in prjtypes.UpdateProjectInput) (*entity.Project, error) {
	return f(t, a, in)
}

func newProjectRouter(m *prjMock) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewProjectHandler(m.useCases())
	r := gin.New()
	g := r.Group("/api/v1", withIdentity(testTenantID, testActorID))
	g.POST("/projects", h.CreateProject)
	g.GET("/projects", h.ListProjects)
	g.GET("/projects/code-suggestion", h.SuggestProjectCode)
	g.GET("/projects/:id", h.GetProject)
	g.PATCH("/projects/:id", h.UpdateProject)
	return r
}

func doProject(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func sampleProject() *entity.Project {
	return &entity.Project{ID: uuid.New(), TenantID: testTenantID, Name: "Apollo", Code: "APO", Status: "active",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func TestProjectHandler_Projects(t *testing.T) {
	p := sampleProject()
	m := &prjMock{
		create: func(tenantID, actorID uuid.UUID, in prjtypes.CreateProjectInput) (*entity.Project, error) {
			assert.Equal(t, testTenantID, tenantID)
			assert.Equal(t, testActorID, actorID)
			if in.Code == "DUP" {
				return nil, domainerrors.ErrProjectCodeTaken
			}
			if in.Code == "BAD" {
				return nil, ErrTestInvalid
			}
			assert.Equal(t, "2026-01-01", in.StartDate.Format("2006-01-02"))
			return p, nil
		},
		get: func(tenantID, id uuid.UUID) (*entity.Project, error) {
			if id != p.ID {
				return nil, domainerrors.ErrProjectNotFound
			}
			return p, nil
		},
		list: func(tenantID uuid.UUID, q prjtypes.ListProjectsQuery) (*prjtypes.ListProjectsResult, error) {
			assert.Equal(t, "apo", q.Search)
			assert.Equal(t, "on_hold", q.Status)
			assert.Equal(t, 2, q.Page)
			return &prjtypes.ListProjectsResult{Projects: []*entity.Project{p}, Total: 41, Page: 2, PageSize: 20}, nil
		},
		suggest: func(tenantID uuid.UUID, name string) (string, error) {
			if name == "" {
				return "", ErrTestInvalid
			}
			return "APOLLO-X", nil
		},
		update: func(tenantID, actorID uuid.UUID, in prjtypes.UpdateProjectInput) (*entity.Project, error) {
			assert.True(t, in.Status.Set)
			assert.True(t, in.EndDate.Set && in.EndDate.Null)
			assert.False(t, in.Name.Set)
			return p, nil
		},
	}
	r := newProjectRouter(m)

	rec := doProject(r, http.MethodPost, "/api/v1/projects", `{"name":"Apollo","code":"apo","start_date":"2026-01-01"}`)
	assert.Equal(t, http.StatusCreated, rec.Code)
	rec = doProject(r, http.MethodPost, "/api/v1/projects", `{"name":"Apollo","code":"DUP","start_date":"2026-01-01"}`)
	assert.Equal(t, http.StatusConflict, rec.Code)
	rec = doProject(r, http.MethodPost, "/api/v1/projects", `{"name":"Apollo","code":"BAD","start_date":"2026-01-01"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	rec = doProject(r, http.MethodPost, "/api/v1/projects", `{"name":"Apollo","code":"X","start_date":"01/02/2026"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "bad date format")
	rec = doProject(r, http.MethodPost, "/api/v1/projects", `{"name":"Apollo","start_date":"2026-01-01"}`)
	assert.Equal(t, http.StatusCreated, rec.Code, "code is optional")
	rec = doProject(r, http.MethodGet, "/api/v1/projects/code-suggestion?name=Apollo+X", "")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"APOLLO-X"`)
	rec = doProject(r, http.MethodGet, "/api/v1/projects/code-suggestion?name=", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	rec = doProject(r, http.MethodPost, "/api/v1/projects", `{`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = doProject(r, http.MethodGet, "/api/v1/projects/"+p.ID.String(), "")
	assert.Equal(t, http.StatusOK, rec.Code)
	rec = doProject(r, http.MethodGet, "/api/v1/projects/"+uuid.NewString(), "")
	assert.Equal(t, http.StatusNotFound, rec.Code, "other tenant's / unknown project")
	rec = doProject(r, http.MethodGet, "/api/v1/projects/not-a-uuid", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = doProject(r, http.MethodGet, "/api/v1/projects?search=apo&status=on_hold&page=2", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var env struct {
		Meta struct {
			TotalItems int64 `json:"total_items"`
			TotalPages int   `json:"total_pages"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.EqualValues(t, 41, env.Meta.TotalItems)
	assert.Equal(t, 3, env.Meta.TotalPages)
	rec = doProject(r, http.MethodGet, "/api/v1/projects?page=x", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = doProject(r, http.MethodPatch, "/api/v1/projects/"+p.ID.String(), `{"status":"archived","end_date":null}`)
	assert.Equal(t, http.StatusOK, rec.Code)
}

var ErrTestInvalid = &testInvalid{}

type testInvalid struct{}

func (*testInvalid) Error() string { return "code: bad" }
func (*testInvalid) Unwrap() error { return domainerrors.ErrInvalidProjectData }

// Without identity in the request context every endpoint is a 401, and a usecase
// ErrForbidden/ErrUnauthorized maps to 403/401; the tenant is never read from the payload.
func TestProjectHandler_AuthContextAndErrorMapping(t *testing.T) {
	called := false
	m := &prjMock{
		create: func(tenantID, actorID uuid.UUID, in prjtypes.CreateProjectInput) (*entity.Project, error) {
			called = true
			return sampleProject(), nil
		},
		get: func(tenantID, id uuid.UUID) (*entity.Project, error) { return nil, domainerrors.ErrForbidden },
		update: func(tenantID, actorID uuid.UUID, in prjtypes.UpdateProjectInput) (*entity.Project, error) {
			return nil, domainerrors.ErrUnauthorized
		},
		list: func(tenantID uuid.UUID, q prjtypes.ListProjectsQuery) (*prjtypes.ListProjectsResult, error) {
			return nil, assert.AnError
		},
	}
	gin.SetMode(gin.TestMode)
	h := NewProjectHandler(m.useCases())
	bare := gin.New() // no identity middleware
	bare.POST("/projects", h.CreateProject)
	bare.GET("/projects", h.ListProjects)
	bare.GET("/projects/:id", h.GetProject)
	bare.PATCH("/projects/:id", h.UpdateProject)
	id := uuid.NewString()
	assert.Equal(t, http.StatusUnauthorized, doProject(bare, http.MethodPost, "/projects", `{"name":"a","code":"b","start_date":"2026-01-01"}`).Code)
	assert.Equal(t, http.StatusUnauthorized, doProject(bare, http.MethodGet, "/projects", "").Code)
	assert.Equal(t, http.StatusUnauthorized, doProject(bare, http.MethodGet, "/projects/"+id, "").Code)
	assert.Equal(t, http.StatusUnauthorized, doProject(bare, http.MethodPatch, "/projects/"+id, `{}`).Code)
	assert.False(t, called, "usecase must not run without an authenticated identity")

	r := newProjectRouter(m)
	assert.Equal(t, http.StatusForbidden, doProject(r, http.MethodGet, "/api/v1/projects/"+id, "").Code)
	assert.Equal(t, http.StatusUnauthorized, doProject(r, http.MethodPatch, "/api/v1/projects/"+id, `{}`).Code)
	assert.Equal(t, http.StatusInternalServerError, doProject(r, http.MethodGet, "/api/v1/projects", "").Code)
}
