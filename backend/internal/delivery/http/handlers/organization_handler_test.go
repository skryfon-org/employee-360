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

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	orgtypes "github.com/skryfon-org/employee-360/backend/internal/types/organization"
	orgimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/organization"
)

type mockGetOrgSettings struct {
	fn func(ctx context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error)
}

func (m mockGetOrgSettings) Execute(ctx context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error) {
	return m.fn(ctx, tenantID)
}

type mockUpdateOrgSettings struct {
	fn func(ctx context.Context, tenantID, actorID uuid.UUID, in orgtypes.UpdateSettingsInput) (*entity.OrganizationSettings, error)
}

func (m mockUpdateOrgSettings) Execute(ctx context.Context, tenantID, actorID uuid.UUID, in orgtypes.UpdateSettingsInput) (*entity.OrganizationSettings, error) {
	return m.fn(ctx, tenantID, actorID, in)
}

func orgRouter(h *OrganizationHandler, withID bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var g *gin.RouterGroup
	if withID {
		g = r.Group("/api/v1/organization/settings", withIdentity(testTenantID, testActorID))
	} else {
		g = r.Group("/api/v1/organization/settings")
	}
	g.GET("", h.GetSettings)
	g.PUT("", h.UpdateSettings)
	return r
}

func orgDo(r *gin.Engine, method, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/organization/settings", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func orgData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var env struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	return env.Data
}

func TestOrganizationHandler_Get(t *testing.T) {
	h := NewOrganizationHandler(mockGetOrgSettings{fn: func(_ context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error) {
		assert.Equal(t, testTenantID, tenantID)
		s := entity.DefaultOrganizationSettings(tenantID)
		s.EmployeeCodePrefix, s.EmployeeCodeStartNumber, s.EmployeeCodeNextNumber, s.EmployeeCodeLastIssuedNumber = "SKRYFON", 40001, 40003, 40002
		s.UpdatedAt = time.Now()
		return s, nil
	}}, nil)
	rec := orgDo(orgRouter(h, true), http.MethodGet, "")
	require.Equal(t, http.StatusOK, rec.Code)
	d := orgData(t, rec)
	assert.Equal(t, "SKRYFON", d["employee_code_prefix"])
	assert.EqualValues(t, 4, d["employee_code_padding"])
	assert.EqualValues(t, 40001, d["employee_code_start_number"])
	assert.EqualValues(t, 40003, d["employee_code_next_number"])
	assert.Equal(t, true, d["employee_codes_issued"])
	assert.Equal(t, "SKRYFON40003", d["employee_code_preview"])
	assert.Equal(t, testTenantID.String(), d["tenant_id"])
}

func TestOrganizationHandler_Get_DefaultsOmitUpdatedAt(t *testing.T) {
	h := NewOrganizationHandler(mockGetOrgSettings{fn: func(_ context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error) {
		return entity.DefaultOrganizationSettings(tenantID), nil
	}}, nil)
	d := orgData(t, orgDo(orgRouter(h, true), http.MethodGet, ""))
	assert.Equal(t, "EMP0001", d["employee_code_preview"])
	assert.Equal(t, false, d["employee_codes_issued"])
	assert.NotContains(t, d, "updated_at")
}

func TestOrganizationHandler_Unauthenticated401(t *testing.T) {
	h := NewOrganizationHandler(nil, nil)
	assert.Equal(t, http.StatusUnauthorized, orgDo(orgRouter(h, false), http.MethodGet, "").Code)
	assert.Equal(t, http.StatusUnauthorized, orgDo(orgRouter(h, false), http.MethodPut, `{}`).Code)
}

func TestOrganizationHandler_Update(t *testing.T) {
	var gotIn orgtypes.UpdateSettingsInput
	h := NewOrganizationHandler(nil, mockUpdateOrgSettings{fn: func(_ context.Context, tenantID, actorID uuid.UUID, in orgtypes.UpdateSettingsInput) (*entity.OrganizationSettings, error) {
		assert.Equal(t, testTenantID, tenantID)
		assert.Equal(t, testActorID, actorID)
		gotIn = in
		s := entity.DefaultOrganizationSettings(tenantID)
		s.EmployeeCodePrefix, s.EmployeeCodePadding, s.EmployeeCodeStartNumber, s.EmployeeCodeNextNumber = "SKRYFON", in.EmployeeCodePadding, in.EmployeeCodeStartNumber, in.EmployeeCodeStartNumber
		return s, nil
	}})
	// A client-supplied tenant_id and counter are ignored.
	body := `{"employee_code_prefix":"skryfon","employee_code_padding":4,"employee_code_start_number":40001,"employee_code_next_number":999,"tenant_id":"` + uuid.NewString() + `"}`
	rec := orgDo(orgRouter(h, true), http.MethodPut, body)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, orgtypes.UpdateSettingsInput{EmployeeCodePrefix: "skryfon", EmployeeCodePadding: 4, EmployeeCodeStartNumber: 40001}, gotIn)
	d := orgData(t, rec)
	assert.Equal(t, "SKRYFON40001", d["employee_code_preview"])
	assert.Equal(t, testTenantID.String(), d["tenant_id"])
}

func TestOrganizationHandler_Update_BadRequests(t *testing.T) {
	called := false
	h := NewOrganizationHandler(nil, mockUpdateOrgSettings{fn: func(context.Context, uuid.UUID, uuid.UUID, orgtypes.UpdateSettingsInput) (*entity.OrganizationSettings, error) {
		called = true
		return nil, nil
	}})
	for name, body := range map[string]string{
		"malformed":     `{`,
		"missing field": `{"employee_code_prefix":"EMP","employee_code_padding":4}`,
		"no prefix":     `{"employee_code_padding":4,"employee_code_start_number":1}`,
		"string number": `{"employee_code_prefix":"EMP","employee_code_padding":"4","employee_code_start_number":1}`,
		"float start":   `{"employee_code_prefix":"EMP","employee_code_padding":4,"employee_code_start_number":1.5}`,
	} {
		rec := orgDo(orgRouter(h, true), http.MethodPut, body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, name)
	}
	assert.False(t, called)
}

func TestOrganizationHandler_Update_ValidationAndErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code int
	}{
		{"prefix ends in digit", orgimpl.ErrPrefixEndsInDigit, http.StatusBadRequest},
		{"padding", orgimpl.ErrPaddingRange, http.StatusBadRequest},
		{"unauthorized", domainerrors.ErrUnauthorized, http.StatusUnauthorized},
		{"unexpected", assert.AnError, http.StatusInternalServerError},
	}
	for _, c := range cases {
		h := NewOrganizationHandler(nil, mockUpdateOrgSettings{fn: func(context.Context, uuid.UUID, uuid.UUID, orgtypes.UpdateSettingsInput) (*entity.OrganizationSettings, error) {
			return nil, c.err
		}})
		rec := orgDo(orgRouter(h, true), http.MethodPut, `{"employee_code_prefix":"EMP","employee_code_padding":4,"employee_code_start_number":1}`)
		assert.Equal(t, c.code, rec.Code, c.name)
		if c.code == http.StatusBadRequest {
			assert.Contains(t, rec.Body.String(), c.err.Error())
		}
	}
}
