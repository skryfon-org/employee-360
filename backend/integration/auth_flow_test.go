//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/config"
	deliveryhttp "github.com/skryfon-org/employee-360/backend/internal/delivery/http"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/eventing"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
)

const authFlowPassword = "S3cret-passw0rd"

type authFlowEnv struct {
	db       *gorm.DB
	router   http.Handler
	tenantID uuid.UUID
	userID   uuid.UUID
	email    string
	domain   string
}

// envelopeBody mirrors response.Envelope for decoding.
type envelopeBody struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

// newAuthFlowEnv boots the real container + router against the live DB and
// seeds a tenant, its email domain, an admin role and an active user with a
// known password. Everything is removed on cleanup (tenant delete cascades).
func newAuthFlowEnv(t *testing.T) *authFlowEnv {
	t.Helper()
	db, _, _ := setup(t) // connects, migrates, skips/fails per repo convention

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		JWT:  config.JWTConfig{Secret: "integration-test-secret-integration-test-secret", AccessExpiry: 15 * time.Minute, RefreshExpiry: time.Hour},
		App:  config.AppConfig{FrontendURL: "http://frontend.test"},
		CORS: config.CORSConfig{AllowedOrigins: []string{"*"}},
	}
	ctr, err := container.New(cfg, db, nil, zerolog.Nop())
	require.NoError(t, err)
	router := deliveryhttp.SetupRouter(cfg, zerolog.Nop(), ctr)

	suffix := uuid.NewString()[:8]
	env := &authFlowEnv{
		db:       db,
		router:   router,
		tenantID: uuid.New(),
		userID:   uuid.New(),
		domain:   "authflow-" + suffix + ".test",
	}
	env.email = "alice@" + env.domain

	hash, err := infraservice.NewHashService().HashPassword(authFlowPassword)
	require.NoError(t, err)
	roleID := uuid.New()

	t.Cleanup(func() {
		db.Exec("DELETE FROM river_job WHERE kind = 'send_email' AND args->>'to' = ?", env.email)
		db.Exec("DELETE FROM tenants WHERE id = ?", env.tenantID)
	})

	require.NoError(t, db.Create(&entity.Tenant{ID: env.tenantID, Name: "authflow-" + suffix, IsActive: true}).Error)
	require.NoError(t, db.Create(&entity.TenantDomain{ID: uuid.New(), TenantID: env.tenantID, Domain: env.domain}).Error)
	require.NoError(t, db.Create(&entity.Role{ID: roleID, TenantID: env.tenantID, Name: entity.RoleAdmin}).Error)
	require.NoError(t, db.Create(&entity.User{
		ID: env.userID, TenantID: env.tenantID, FirstName: "Alice", LastName: "Auth",
		Email: env.email, PasswordHash: &hash, IsActive: true,
	}).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO user_roles (id, tenant_id, user_id, role_id) VALUES (?, ?, ?, ?)",
		uuid.New(), env.tenantID, env.userID, roleID).Error)
	return env
}

func (e *authFlowEnv) post(t *testing.T, path string, body any, bearer string) (int, envelopeBody, []byte) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth"+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	var env envelopeBody
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	return rec.Code, env, rec.Body.Bytes()
}

type tokenPairBody struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (e *authFlowEnv) login(t *testing.T, password string) tokenPairBody {
	t.Helper()
	code, env, raw := e.post(t, "/login", map[string]string{"email": e.email, "password": password}, "")
	require.Equal(t, http.StatusOK, code, string(raw))
	var pair tokenPairBody
	require.NoError(t, json.Unmarshal(env.Data, &pair))
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
	return pair
}

func (e *authFlowEnv) activeRefreshTokens(t *testing.T) int64 {
	t.Helper()
	var n int64
	require.NoError(t, e.db.Raw(
		"SELECT count(*) FROM refresh_tokens WHERE tenant_id = ? AND user_id = ? AND revoked_at IS NULL",
		e.tenantID, e.userID).Scan(&n).Error)
	return n
}

// TestAuthFlow_LoginRefreshLogout drives login -> refresh (rotation) -> logout
// (revocation) through the real router, container and live database.
func TestAuthFlow_LoginRefreshLogout(t *testing.T) {
	e := newAuthFlowEnv(t)

	// Wrong password and unknown domain are indistinguishable 401s.
	code, _, _ := e.post(t, "/login", map[string]string{"email": e.email, "password": "wrong-password"}, "")
	require.Equal(t, http.StatusUnauthorized, code)
	code, _, _ = e.post(t, "/login", map[string]string{"email": "alice@unknown-" + e.domain, "password": authFlowPassword}, "")
	require.Equal(t, http.StatusUnauthorized, code)

	first := e.login(t, authFlowPassword)
	require.EqualValues(t, 1, e.activeRefreshTokens(t))

	// Refresh rotates: new pair, old refresh token no longer usable.
	code, env, raw := e.post(t, "/refresh", map[string]string{"refresh_token": first.RefreshToken}, "")
	require.Equal(t, http.StatusOK, code, string(raw))
	var second tokenPairBody
	require.NoError(t, json.Unmarshal(env.Data, &second))
	require.NotEmpty(t, second.AccessToken)
	require.NotEqual(t, first.RefreshToken, second.RefreshToken, "refresh token must rotate")

	code, _, _ = e.post(t, "/refresh", map[string]string{"refresh_token": first.RefreshToken}, "")
	require.Equal(t, http.StatusUnauthorized, code, "reusing a rotated refresh token must fail")

	// Reuse of a rotated token revokes the whole family (theft detection), so
	// log in again for a clean session before testing logout.
	third := e.login(t, authFlowPassword)

	// Logout requires a Bearer access token.
	code, _, _ = e.post(t, "/logout", map[string]string{"refresh_token": third.RefreshToken}, "")
	require.Equal(t, http.StatusUnauthorized, code)

	code, _, raw = e.post(t, "/logout", map[string]string{"refresh_token": third.RefreshToken}, third.AccessToken)
	require.Equal(t, http.StatusOK, code, string(raw))

	code, _, _ = e.post(t, "/refresh", map[string]string{"refresh_token": third.RefreshToken}, "")
	require.Equal(t, http.StatusUnauthorized, code, "refresh with a revoked token must fail")
}

// TestAuthFlow_ForgotPasswordWorkerReset proves forgot-password enqueues a
// River job atomically, is enumeration-safe, and that the emailed token resets
// the password and revokes every refresh token.
func TestAuthFlow_ForgotPasswordWorkerReset(t *testing.T) {
	e := newAuthFlowEnv(t)
	sqlDB, err := e.db.DB()
	require.NoError(t, err)

	var pending int64
	require.NoError(t, e.db.Raw("SELECT count(*) FROM river_job WHERE kind = 'send_email' AND state IN ('available','retryable','scheduled')").Scan(&pending).Error)
	if pending > 0 {
		t.Skipf("skipping: %d unrelated pending send_email jobs would be consumed by the test worker", pending)
	}

	// An existing session that the reset must revoke.
	old := e.login(t, authFlowPassword)
	e.login(t, authFlowPassword) // second concurrent session
	require.EqualValues(t, 2, e.activeRefreshTokens(t))

	// Unknown email in a known domain and unknown domain: identical response, no job.
	unknown := "nobody@" + e.domain
	codeK, _, rawKnown := e.post(t, "/forgot-password", map[string]string{"email": e.email}, "")
	codeU, _, rawUnknown := e.post(t, "/forgot-password", map[string]string{"email": unknown}, "")
	codeD, _, rawDomain := e.post(t, "/forgot-password", map[string]string{"email": "x@nope-" + e.domain}, "")
	require.Equal(t, http.StatusOK, codeK)
	require.Equal(t, codeK, codeU)
	require.Equal(t, codeK, codeD)
	require.Equal(t, string(rawKnown), string(rawUnknown), "response must not reveal whether the email exists")
	require.Equal(t, string(rawKnown), string(rawDomain))

	// Job enqueued in the same transaction as the reset token.
	require.EqualValues(t, 1, jobCount(t, e.db, e.email))
	require.EqualValues(t, 0, jobCount(t, e.db, unknown))
	var tokens int64
	require.NoError(t, e.db.Raw("SELECT count(*) FROM password_reset_tokens WHERE tenant_id = ? AND user_id = ? AND used_at IS NULL", e.tenantID, e.userID).Scan(&tokens).Error)
	require.EqualValues(t, 1, tokens)

	// Run the worker with a recording EmailService and pick up the reset link.
	rec := &recordingEmailService{}
	worker, err := eventing.NewWorkerClient(sqlDB, rec, nil, 2)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, worker.Start(ctx))
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		_ = worker.Stop(stopCtx)
	})
	require.Eventually(t, func() bool { return len(rec.to(e.email)) == 1 }, 15*time.Second, 200*time.Millisecond)

	resetURL, ok := rec.to(e.email)[0].TemplateData["reset_url"].(string)
	require.True(t, ok, "reset_url template datum must be a string")
	parsed, err := url.Parse(resetURL)
	require.NoError(t, err)
	token := parsed.Query().Get("token")
	require.NotEmpty(t, token)

	// Bad token and too-short password are rejected without side effects.
	code, _, _ := e.post(t, "/reset-password", map[string]string{"token": "deadbeef", "new_password": "another-passw0rd"}, "")
	require.Equal(t, http.StatusBadRequest, code)
	code, _, _ = e.post(t, "/reset-password", map[string]string{"token": token, "new_password": "short"}, "")
	require.Equal(t, http.StatusBadRequest, code)
	require.EqualValues(t, 2, e.activeRefreshTokens(t))

	const newPassword = "brand-new-passw0rd"
	code, _, raw := e.post(t, "/reset-password", map[string]string{"token": token, "new_password": newPassword}, "")
	require.Equal(t, http.StatusOK, code, string(raw))

	// All refresh tokens revoked; the old one no longer refreshes.
	require.EqualValues(t, 0, e.activeRefreshTokens(t), "reset must revoke all refresh tokens")
	code, _, _ = e.post(t, "/refresh", map[string]string{"refresh_token": old.RefreshToken}, "")
	require.Equal(t, http.StatusUnauthorized, code)

	// Token is single-use; old password fails, new one works.
	code, _, _ = e.post(t, "/reset-password", map[string]string{"token": token, "new_password": "yet-another-passw0rd"}, "")
	require.Equal(t, http.StatusBadRequest, code)
	code, _, _ = e.post(t, "/login", map[string]string{"email": e.email, "password": authFlowPassword}, "")
	require.Equal(t, http.StatusUnauthorized, code)
	e.login(t, newPassword)
}
