package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	tenanttypes "github.com/skryfon-org/employee-360/backend/internal/types/tenant"
)

type tenantFakes struct {
	err error

	gotTenantID uuid.UUID
	gotActor    uuid.UUID
	gotDomainID uuid.UUID
	gotName     string
	gotDomain   string
	calls       int
}

func (f *tenantFakes) tenant(id uuid.UUID) *entity.Tenant {
	uid := uuid.New()
	return &entity.Tenant{ID: id, Name: "Acme", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), CreatedBy: &uid, UpdatedBy: &uid}
}

type fGet struct{ f *tenantFakes }
type fRename struct{ f *tenantFakes }
type fListDomains struct{ f *tenantFakes }
type fAddDomain struct{ f *tenantFakes }
type fUpdDomain struct{ f *tenantFakes }
type fDelDomain struct{ f *tenantFakes }

func (x fGet) Execute(_ context.Context, tid uuid.UUID) (*tenanttypes.TenantDetail, error) {
	x.f.calls++
	x.f.gotTenantID = tid
	if x.f.err != nil {
		return nil, x.f.err
	}
	creator := uuid.New()
	return &tenanttypes.TenantDetail{Tenant: x.f.tenant(tid), Domains: []*entity.TenantDomain{{ID: uuid.New(), TenantID: tid, Domain: "acme.com", CreatedBy: &creator}}}, nil
}
func (x fRename) Execute(_ context.Context, actor, tid uuid.UUID, req tenanttypes.UpdateTenantRequest) (*entity.Tenant, error) {
	x.f.calls++
	x.f.gotActor, x.f.gotTenantID, x.f.gotName = actor, tid, req.Name
	if x.f.err != nil {
		return nil, x.f.err
	}
	return x.f.tenant(tid), nil
}
func (x fListDomains) Execute(_ context.Context, tid uuid.UUID) ([]*entity.TenantDomain, error) {
	x.f.calls++
	x.f.gotTenantID = tid
	if x.f.err != nil {
		return nil, x.f.err
	}
	creator := uuid.New()
	return []*entity.TenantDomain{{ID: uuid.New(), TenantID: tid, Domain: "acme.com", CreatedBy: &creator}}, nil
}
func (x fAddDomain) Execute(_ context.Context, actor, tid uuid.UUID, req tenanttypes.AddDomainRequest) (*entity.TenantDomain, error) {
	x.f.calls++
	x.f.gotActor, x.f.gotTenantID, x.f.gotDomain = actor, tid, req.Domain
	if x.f.err != nil {
		return nil, x.f.err
	}
	return &entity.TenantDomain{ID: uuid.New(), TenantID: tid, Domain: req.Domain}, nil
}
func (x fUpdDomain) Execute(_ context.Context, actor, tid, domainID uuid.UUID, req tenanttypes.UpdateDomainRequest) (*entity.TenantDomain, error) {
	x.f.calls++
	x.f.gotActor, x.f.gotTenantID, x.f.gotDomainID, x.f.gotDomain = actor, tid, domainID, req.Domain
	if x.f.err != nil {
		return nil, x.f.err
	}
	return &entity.TenantDomain{ID: domainID, TenantID: tid, Domain: req.Domain}, nil
}
func (x fDelDomain) Execute(_ context.Context, actor, tid, domainID uuid.UUID) error {
	x.f.calls++
	x.f.gotActor, x.f.gotTenantID, x.f.gotDomainID = actor, tid, domainID
	return x.f.err
}

func tenantEngine(f *tenantFakes, mw ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewTenantHandler(TenantUseCases{
		Get: fGet{f}, Rename: fRename{f}, ListDomains: fListDomains{f}, AddDomain: fAddDomain{f},
		UpdateDomain: fUpdDomain{f}, DelDomain: fDelDomain{f},
	})
	e := gin.New()
	e.Use(mw...)
	e.GET("/tenant", h.Get)
	e.PATCH("/tenant", h.Update)
	e.GET("/tenant/domains", h.ListDomains)
	e.POST("/tenant/domains", h.AddDomain)
	e.PATCH("/tenant/domains/:domainId", h.UpdateDomain)
	e.DELETE("/tenant/domains/:domainId", h.RemoveDomain)
	return e
}

func tcall(e *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var b struct {
		Error struct{ Code string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &b))
	return b.Error.Code
}

func keys(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestTenantHandler_Get_TenantFromContext_NoInternalFields(t *testing.T) {
	tid := uuid.New()
	f := &tenantFakes{}
	e := tenantEngine(f, withIdentity(tid, uuid.New()))

	// A tenant id in the query string is ignored.
	rec := tcall(e, http.MethodGet, "/tenant?tenant_id="+uuid.New().String()+"&id="+uuid.New().String(), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, tid, f.gotTenantID)
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.ElementsMatch(t, []string{"id", "name", "is_active", "created_at", "updated_at", "domains"}, keys(body.Data))
	doms := body.Data["domains"].([]any)
	require.Len(t, doms, 1)
	assert.ElementsMatch(t, []string{"id", "tenant_id", "domain", "created_at"}, keys(doms[0].(map[string]any)))
}

func TestTenantHandler_Rename(t *testing.T) {
	tid, actor := uuid.New(), uuid.New()
	f := &tenantFakes{}
	e := tenantEngine(f, withIdentity(tid, actor))

	rec := tcall(e, http.MethodPatch, "/tenant", `{"name":"New","tenant_id":"`+uuid.New().String()+`","id":"`+uuid.New().String()+`","actor_id":"`+uuid.New().String()+`"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "New", f.gotName)
	assert.Equal(t, tid, f.gotTenantID, "body tenant ignored")
	assert.Equal(t, actor, f.gotActor)
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.ElementsMatch(t, []string{"id", "name", "is_active", "created_at", "updated_at"}, keys(body.Data))

	assert.Equal(t, http.StatusBadRequest, tcall(e, http.MethodPatch, "/tenant", `{`).Code)
}

func TestTenantHandler_Domains(t *testing.T) {
	tid, actor := uuid.New(), uuid.New()
	f := &tenantFakes{}
	e := tenantEngine(f, withIdentity(tid, actor))

	rec := tcall(e, http.MethodGet, "/tenant/domains", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, tid, f.gotTenantID)
	assert.NotContains(t, rec.Body.String(), "created_by")

	rec = tcall(e, http.MethodPost, "/tenant/domains", `{"domain":"new.com","tenant_id":"`+uuid.New().String()+`"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "new.com", f.gotDomain)
	assert.Equal(t, tid, f.gotTenantID)
	assert.Equal(t, actor, f.gotActor)
	assert.Equal(t, http.StatusBadRequest, tcall(e, http.MethodPost, "/tenant/domains", `{`).Code)

	dom := uuid.New()
	rec = tcall(e, http.MethodPatch, "/tenant/domains/"+dom.String(), `{"domain":"upd.com","tenant_id":"`+uuid.New().String()+`"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, dom, f.gotDomainID)
	assert.Equal(t, "upd.com", f.gotDomain)
	assert.Equal(t, tid, f.gotTenantID)
	assert.Equal(t, actor, f.gotActor)
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.ElementsMatch(t, []string{"id", "tenant_id", "domain", "created_at"}, keys(body.Data))
	assert.Equal(t, http.StatusBadRequest, tcall(e, http.MethodPatch, "/tenant/domains/"+dom.String(), `{`).Code)

	dom = uuid.New()
	rec = tcall(e, http.MethodDelete, "/tenant/domains/"+dom.String(), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, dom, f.gotDomainID)
	assert.Equal(t, tid, f.gotTenantID)
	assert.Equal(t, actor, f.gotActor)
}

func TestTenantHandler_InvalidDomainID(t *testing.T) {
	f := &tenantFakes{}
	e := tenantEngine(f, withIdentity(uuid.New(), uuid.New()))
	for _, id := range []string{"nope", uuid.Nil.String()} {
		assert.Equal(t, http.StatusBadRequest, tcall(e, http.MethodPatch, "/tenant/domains/"+id, `{"domain":"x.com"}`).Code, id)
		assert.Equal(t, http.StatusBadRequest, tcall(e, http.MethodDelete, "/tenant/domains/"+id, "").Code, id)
	}
	assert.Zero(t, f.calls)
}

func TestTenantHandler_ErrorMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{domainerrors.ErrDomainAlreadyExists, 409, "DOMAIN_ALREADY_EXISTS"},
		{domainerrors.ErrLastDomain, 409, "LAST_DOMAIN"},
		{domainerrors.ErrDomainInUse, 409, "DOMAIN_IN_USE"},
		{domainerrors.ErrInvalidDomain, 400, "INVALID_DOMAIN"},
		{domainerrors.ErrInvalidTenantName, 400, "INVALID_NAME"},
		{domainerrors.ErrTenantNotFound, 404, "NOT_FOUND"},
		{domainerrors.ErrDomainNotFound, 404, "NOT_FOUND"},
		{domainerrors.ErrUnauthorized, 401, "UNAUTHORIZED"},
		{errors.New("db exploded"), 500, "INTERNAL_ERROR"},
	}
	dom := uuid.New().String()
	for _, tc := range cases {
		f := &tenantFakes{err: tc.err}
		e := tenantEngine(f, withIdentity(uuid.New(), uuid.New()))
		for _, r := range [][3]string{
			{http.MethodPost, "/tenant/domains", `{"domain":"x.com"}`},
			{http.MethodPatch, "/tenant/domains/" + dom, `{"domain":"x.com"}`},
			{http.MethodDelete, "/tenant/domains/" + dom, ""},
			{http.MethodPatch, "/tenant", `{"name":"x"}`},
			{http.MethodGet, "/tenant", ""},
			{http.MethodGet, "/tenant/domains", ""},
		} {
			rec := tcall(e, r[0], r[1], r[2])
			assert.Equal(t, tc.status, rec.Code, "%s %s %v", r[0], r[1], tc.err)
			assert.Equal(t, tc.code, errCode(t, rec), tc.err)
			assert.NotContains(t, rec.Body.String(), "db exploded")
		}
	}
}

func TestTenantHandler_MissingIdentityIs401(t *testing.T) {
	f := &tenantFakes{}
	e := tenantEngine(f) // no identity middleware
	dom := uuid.New().String()
	for _, r := range [][3]string{
		{http.MethodGet, "/tenant", ""},
		{http.MethodPatch, "/tenant", `{"name":"A"}`},
		{http.MethodGet, "/tenant/domains", ""},
		{http.MethodPost, "/tenant/domains", `{"domain":"a.com"}`},
		{http.MethodPatch, "/tenant/domains/" + dom, `{"domain":"a.com"}`},
		{http.MethodDelete, "/tenant/domains/" + dom, ""},
	} {
		assert.Equal(t, http.StatusUnauthorized, tcall(e, r[0], r[1], r[2]).Code, r[0]+" "+r[1])
	}
	assert.Zero(t, f.calls, "no usecase runs without identity")
}
