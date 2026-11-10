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

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/eventing"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
)

// doJSON sends a JSON request to an arbitrary /api/v1 path on the env router.
func (e *authFlowEnv) doJSON(t *testing.T, method, path string, body any, bearer string) (int, envelopeBody, []byte) {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(raw)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
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

// TestInvitationFlow_InviteDeliverAccept drives the full onboarding path through
// the real router, container, River outbox, worker and a capturing mailer:
// admin invites -> email delivered -> token extracted -> unauthenticated accept
// -> the user is active with a password and can log in.
func TestInvitationFlow_InviteDeliverAccept(t *testing.T) {
	e := newAuthFlowEnv(t)
	sqlDB, err := e.db.DB()
	require.NoError(t, err)

	var pending int64
	require.NoError(t, e.db.Raw("SELECT count(*) FROM river_job WHERE kind = 'send_email' AND state IN ('available','retryable','scheduled')").Scan(&pending).Error)
	if pending > 0 {
		t.Skipf("skipping: %d unrelated pending send_email jobs would be consumed by the test worker", pending)
	}

	// Employee role plus a non-admin user for the 403 check.
	employeeRoleID := uuid.New()
	require.NoError(t, e.db.Create(&entity.Role{ID: employeeRoleID, TenantID: e.tenantID, Name: entity.RoleEmployee}).Error)
	empEmail := "bob@" + e.domain
	empHash, err := infraservice.NewHashService().HashPassword(authFlowPassword)
	require.NoError(t, err)
	empID := uuid.New()
	require.NoError(t, e.db.Create(&entity.User{
		ID: empID, TenantID: e.tenantID, FirstName: "Bob", LastName: "Employee",
		Email: empEmail, PasswordHash: &empHash, IsActive: true,
	}).Error)
	require.NoError(t, e.db.Exec("INSERT INTO user_roles (id, tenant_id, user_id, role_id) VALUES (?, ?, ?, ?)",
		uuid.New(), e.tenantID, empID, employeeRoleID).Error)

	adminTok := e.login(t, authFlowPassword).AccessToken
	code, env, raw := e.post(t, "/login", map[string]string{"email": empEmail, "password": authFlowPassword}, "")
	require.Equal(t, http.StatusOK, code, string(raw))
	var empPair tokenPairBody
	require.NoError(t, json.Unmarshal(env.Data, &empPair))

	inviteeEmail := "carol@" + e.domain
	t.Cleanup(func() {
		e.db.Exec("DELETE FROM river_job WHERE kind = 'send_email' AND args->>'to' = ?", inviteeEmail)
	})
	body := map[string]any{"email": inviteeEmail, "role_id": employeeRoleID, "first_name": "Carol", "last_name": "Invitee"}

	// Non-admin and unauthenticated callers cannot invite.
	code, _, _ = e.doJSON(t, http.MethodPost, "/api/v1/users/invitations", body, empPair.AccessToken)
	require.Equal(t, http.StatusForbidden, code, "employee token must get 403 on invite")
	code, _, _ = e.doJSON(t, http.MethodPost, "/api/v1/users/invitations", body, "")
	require.Equal(t, http.StatusUnauthorized, code)
	require.EqualValues(t, 0, jobCount(t, e.db, inviteeEmail))

	// Admin invites.
	code, _, raw = e.doJSON(t, http.MethodPost, "/api/v1/users/invitations", body, adminTok)
	require.Equal(t, http.StatusCreated, code, string(raw))
	require.NotContains(t, string(raw), "token", "response must never expose the token")
	require.EqualValues(t, 1, jobCount(t, e.db, inviteeEmail), "invite email must be enqueued with the invitation")

	// Invitee exists but is not active and has no password yet.
	var u entity.User
	require.NoError(t, e.db.Where("tenant_id = ? AND email = ?", e.tenantID, inviteeEmail).First(&u).Error)
	require.False(t, u.IsActive)
	require.Nil(t, u.PasswordHash)

	// Worker delivers to the capturing mailer.
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
	require.Eventually(t, func() bool { return len(rec.to(inviteeEmail)) == 1 }, 15*time.Second, 200*time.Millisecond)

	inviteURL, ok := rec.to(inviteeEmail)[0].TemplateData["invite_url"].(string)
	require.True(t, ok, "invite_url template datum must be a string")
	parsed, err := url.Parse(inviteURL)
	require.NoError(t, err)
	token := parsed.Query().Get("token")
	require.NotEmpty(t, token)

	// Accept is unauthenticated; bad token and weak password are 400 with no side effects.
	code, _, _ = e.doJSON(t, http.MethodPost, "/api/v1/invitations/accept", map[string]string{"token": "deadbeef", "password": "invitee-passw0rd"}, "")
	require.Equal(t, http.StatusBadRequest, code)
	code, _, _ = e.doJSON(t, http.MethodPost, "/api/v1/invitations/accept", map[string]string{"token": token, "password": "short"}, "")
	require.Equal(t, http.StatusBadRequest, code)

	const inviteePassword = "invitee-passw0rd"
	code, _, raw = e.doJSON(t, http.MethodPost, "/api/v1/invitations/accept", map[string]string{"token": token, "password": inviteePassword}, "")
	require.Equal(t, http.StatusOK, code, string(raw))

	var after entity.User
	require.NoError(t, e.db.Where("tenant_id = ? AND email = ?", e.tenantID, inviteeEmail).First(&after).Error)
	require.True(t, after.IsActive, "accepted user must be active")
	require.NotNil(t, after.PasswordHash)
	require.NotEmpty(t, *after.PasswordHash)

	// Accepting enqueued a Welcome email in the same transaction; the worker delivers it.
	require.Eventually(t, func() bool { return len(rec.to(inviteeEmail)) == 2 }, 15*time.Second, 200*time.Millisecond)
	var welcome *service.EmailMessage
	for _, m := range rec.to(inviteeEmail) {
		if m.TemplateName == service.EmailTemplateWelcome {
			m := m
			welcome = &m
		}
	}
	require.NotNil(t, welcome, "a Welcome email must be delivered after accept")
	require.Contains(t, welcome.TemplateData["login_url"], "/login")

	// Token is single-use and the new credentials log in.
	// Re-using an accepted token is a distinct 409 INVITATION_ACCEPTED, not a generic 400.
	code, _, raw = e.doJSON(t, http.MethodPost, "/api/v1/invitations/accept", map[string]string{"token": token, "password": "another-passw0rd"}, "")
	require.Equal(t, http.StatusConflict, code, string(raw))
	require.Contains(t, string(raw), "INVITATION_ACCEPTED")
	code, _, raw = e.post(t, "/login", map[string]string{"email": inviteeEmail, "password": inviteePassword}, "")
	require.Equal(t, http.StatusOK, code, string(raw))
}
