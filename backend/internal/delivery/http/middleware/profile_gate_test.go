package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/ctx"
)

type fakeGate struct {
	complete map[uuid.UUID]bool
	err      error
	calls    int
	tenant   uuid.UUID
}

func (f *fakeGate) Execute(_ context.Context, tenantID, userID uuid.UUID) (bool, error) {
	f.calls++
	f.tenant = tenantID
	return f.complete[userID], f.err
}

func gateEngine(gate *fakeGate, tenant, user uuid.UUID, roles ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyUserID, user.String())
		c.Set(ContextKeyRoles, roles)
		rc := ctx.WithTenantID(c.Request.Context(), tenant.String())
		c.Request = c.Request.WithContext(ctx.WithUserID(rc, user.String()))
	})
	r.GET("/x", RequireCompleteProfile(gate), func(c *gin.Context) { c.String(200, "ok") })
	return r
}

func gateHit(r *gin.Engine) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	return rec
}

func TestRequireCompleteProfile(t *testing.T) {
	tenant, user := uuid.New(), uuid.New()

	t.Run("incomplete employee gets 403 profile_incomplete", func(t *testing.T) {
		gate := &fakeGate{complete: map[uuid.UUID]bool{}}
		rec := gateHit(gateEngine(gate, tenant, user, "employee"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), `"code":"profile_incomplete"`)
		assert.Equal(t, tenant, gate.tenant)
	})
	t.Run("complete employee passes", func(t *testing.T) {
		gate := &fakeGate{complete: map[uuid.UUID]bool{user: true}}
		assert.Equal(t, http.StatusOK, gateHit(gateEngine(gate, tenant, user, "employee")).Code)
	})
	t.Run("status flips once completed", func(t *testing.T) {
		gate := &fakeGate{complete: map[uuid.UUID]bool{}}
		r := gateEngine(gate, tenant, user, "employee")
		assert.Equal(t, http.StatusForbidden, gateHit(r).Code)
		gate.complete[user] = true
		assert.Equal(t, http.StatusOK, gateHit(r).Code)
	})
	t.Run("admins and super admins are never gated", func(t *testing.T) {
		for _, role := range []string{"admin", "super_admin"} {
			gate := &fakeGate{complete: map[uuid.UUID]bool{}}
			assert.Equal(t, http.StatusOK, gateHit(gateEngine(gate, tenant, user, role)).Code, role)
			assert.Zero(t, gate.calls, "no lookup for %s", role)
		}
	})
	t.Run("lookup failure fails closed", func(t *testing.T) {
		gate := &fakeGate{err: errors.New("db down")}
		assert.Equal(t, http.StatusServiceUnavailable, gateHit(gateEngine(gate, tenant, user, "employee")).Code)
	})
	t.Run("missing identity is 401", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/x", RequireCompleteProfile(&fakeGate{}), func(c *gin.Context) { c.String(200, "ok") })
		assert.Equal(t, http.StatusUnauthorized, gateHit(r).Code)
	})
	t.Run("nil gate panics at wiring time", func(t *testing.T) {
		require.Panics(t, func() { RequireCompleteProfile(nil) })
	})
}
