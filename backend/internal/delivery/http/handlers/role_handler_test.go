package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeListRolesUC struct {
	roles     []*entity.Role
	err       error
	gotTenant *uuid.UUID
}

func (f fakeListRolesUC) Execute(_ context.Context, actor emptypes.Actor) ([]*entity.Role, error) {
	*f.gotTenant = actor.TenantID
	return f.roles, f.err
}

func serveRoles(uc fakeListRolesUC) *httptest.ResponseRecorder {
	return serveRolesWith(uc, withIdentity(uuid.New(), uuid.New()))
}

func serveRolesWith(uc fakeListRolesUC, mw ...gin.HandlerFunc) *httptest.ResponseRecorder {
	if uc.gotTenant == nil {
		uc.gotTenant = new(uuid.UUID)
	}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(mw...)
	engine.GET("/roles", NewRoleHandler(uc).List)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/roles", nil))
	return rec
}

func TestRoleHandler_List_OnlyIDAndName(t *testing.T) {
	id := uuid.New()
	rec := serveRoles(fakeListRolesUC{roles: []*entity.Role{{ID: id, TenantID: uuid.New(), Name: "employee"}}})
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Success bool             `json:"success"`
		Data    []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.True(t, body.Success)
	require.Len(t, body.Data, 1)
	assert.Equal(t, map[string]any{"id": id.String(), "name": "employee"}, body.Data[0])
}

func TestRoleHandler_List_EmptyIsArray(t *testing.T) {
	rec := serveRoles(fakeListRolesUC{})
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"data":[]`)
}

func TestRoleHandler_List_Errors(t *testing.T) {
	tests := map[string]struct {
		err  error
		code int
	}{
		"unauthorized": {domainerrors.ErrUnauthorized, http.StatusUnauthorized},
		"forbidden":    {domainerrors.ErrForbidden, http.StatusForbidden},
		"internal":     {errors.New("db down"), http.StatusInternalServerError},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.code, serveRoles(fakeListRolesUC{err: tc.err}).Code)
		})
	}
}

func TestRoleHandler_List_PassesTenantFromContext(t *testing.T) {
	tenant := uuid.New()
	var got uuid.UUID
	rec := serveRolesWith(fakeListRolesUC{gotTenant: &got}, withIdentity(tenant, uuid.New()))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, tenant, got)
}

func TestRoleHandler_List_MissingTenantIs401(t *testing.T) {
	var got uuid.UUID
	rec := serveRolesWith(fakeListRolesUC{gotTenant: &got}) // no identity middleware
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, uuid.Nil, got, "usecase must not be called")
}
