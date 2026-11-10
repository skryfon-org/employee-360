package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/ctx"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	authtypes "github.com/skryfon-org/employee-360/backend/internal/types/auth"
)

type mockLoginUseCase struct {
	executeFn func(ctx context.Context, req authtypes.LoginRequest, ipAddress, userAgent string) (*authtypes.LoginResponse, error)
}

func (m *mockLoginUseCase) Execute(ctx context.Context, req authtypes.LoginRequest, ipAddress, userAgent string) (*authtypes.LoginResponse, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, req, ipAddress, userAgent)
	}
	return nil, nil
}

type mockTokenRefreshUseCase struct {
	executeFn func(ctx context.Context, req authtypes.TokenRefreshRequest, ipAddress, userAgent string) (*authtypes.TokenRefreshResponse, error)
}

func (m *mockTokenRefreshUseCase) Execute(ctx context.Context, req authtypes.TokenRefreshRequest, ipAddress, userAgent string) (*authtypes.TokenRefreshResponse, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, req, ipAddress, userAgent)
	}
	return nil, nil
}

type mockLogoutUseCase struct {
	executeFn func(ctx context.Context, tenantID, userID uuid.UUID, req authtypes.LogoutRequest) error
}

func (m *mockLogoutUseCase) Execute(ctx context.Context, tenantID, userID uuid.UUID, req authtypes.LogoutRequest) error {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, userID, req)
	}
	return nil
}

type mockForgotPasswordUseCase struct {
	executeFn func(ctx context.Context, req authtypes.ForgotPasswordRequest) error
}

func (m *mockForgotPasswordUseCase) Execute(ctx context.Context, req authtypes.ForgotPasswordRequest) error {
	if m.executeFn != nil {
		return m.executeFn(ctx, req)
	}
	return nil
}

type mockResetPasswordUseCase struct {
	executeFn func(ctx context.Context, req authtypes.ResetPasswordRequest) error
}

func (m *mockResetPasswordUseCase) Execute(ctx context.Context, req authtypes.ResetPasswordRequest) error {
	if m.executeFn != nil {
		return m.executeFn(ctx, req)
	}
	return nil
}

func setupAuthHandlerTest() (*gin.Engine, *AuthHandler, *mockLoginUseCase, *mockTokenRefreshUseCase, *mockLogoutUseCase, *mockForgotPasswordUseCase, *mockResetPasswordUseCase) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	loginUC := &mockLoginUseCase{}
	refreshUC := &mockTokenRefreshUseCase{}
	logoutUC := &mockLogoutUseCase{}
	forgotUC := &mockForgotPasswordUseCase{}
	resetUC := &mockResetPasswordUseCase{}

	handler := NewAuthHandler(loginUC, refreshUC, logoutUC, forgotUC, resetUC)

	return engine, handler, loginUC, refreshUC, logoutUC, forgotUC, resetUC
}

func TestAuthHandler_Login(t *testing.T) {
	t.Run("successful login", func(t *testing.T) {
		engine, handler, loginUC, _, _, _, _ := setupAuthHandlerTest()
		engine.POST("/login", handler.Login)

		tenantID := uuid.New()
		userID := uuid.New()
		loginUC.executeFn = func(ctx context.Context, req authtypes.LoginRequest, ipAddress, userAgent string) (*authtypes.LoginResponse, error) {
			if req.Email != "alice@example.com" || req.Password != "password123" {
				t.Errorf("unexpected credentials: %+v", req)
			}
			return &authtypes.LoginResponse{
				AccessToken:  "access-token-123",
				RefreshToken: "refresh-token-123",
				ExpiresAt:    time.Now().Add(15 * time.Minute),
				TokenType:    "Bearer",
				User: &entity.User{
					ID:       userID,
					TenantID: tenantID,
					Email:    "alice@example.com",
				},
			}, nil
		}

		body, _ := json.Marshal(authtypes.LoginRequest{
			Email:    "alice@example.com",
			Password: "password123",
		})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp response.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success=true")
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		engine, handler, loginUC, _, _, _, _ := setupAuthHandlerTest()
		engine.POST("/login", handler.Login)

		loginUC.executeFn = func(ctx context.Context, req authtypes.LoginRequest, ipAddress, userAgent string) (*authtypes.LoginResponse, error) {
			return nil, domainerrors.ErrInvalidCredentials
		}

		body, _ := json.Marshal(authtypes.LoginRequest{
			Email:    "alice@example.com",
			Password: "wrong",
		})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("inactive user", func(t *testing.T) {
		engine, handler, loginUC, _, _, _, _ := setupAuthHandlerTest()
		engine.POST("/login", handler.Login)

		loginUC.executeFn = func(ctx context.Context, req authtypes.LoginRequest, ipAddress, userAgent string) (*authtypes.LoginResponse, error) {
			return nil, domainerrors.ErrUserInactive
		}

		body, _ := json.Marshal(authtypes.LoginRequest{
			Email:    "inactive@example.com",
			Password: "password123",
		})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", rec.Code)
		}
	})

	t.Run("invalid json payload", func(t *testing.T) {
		engine, handler, _, _, _, _, _ := setupAuthHandlerTest()
		engine.POST("/login", handler.Login)

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("{invalid-json")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("unexpected usecase error", func(t *testing.T) {
		engine, handler, loginUC, _, _, _, _ := setupAuthHandlerTest()
		engine.POST("/login", handler.Login)

		loginUC.executeFn = func(ctx context.Context, req authtypes.LoginRequest, ipAddress, userAgent string) (*authtypes.LoginResponse, error) {
			return nil, errors.New("db failure")
		}

		body, _ := json.Marshal(authtypes.LoginRequest{
			Email:    "alice@example.com",
			Password: "password123",
		})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	t.Run("successful refresh", func(t *testing.T) {
		engine, handler, _, refreshUC, _, _, _ := setupAuthHandlerTest()
		engine.POST("/refresh", handler.Refresh)

		refreshUC.executeFn = func(ctx context.Context, req authtypes.TokenRefreshRequest, ipAddress, userAgent string) (*authtypes.TokenRefreshResponse, error) {
			if req.RefreshToken != "valid-refresh-token" {
				t.Errorf("unexpected refresh token: %s", req.RefreshToken)
			}
			return &authtypes.TokenRefreshResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresAt:    time.Now().Add(15 * time.Minute),
				TokenType:    "Bearer",
			}, nil
		}

		body, _ := json.Marshal(authtypes.TokenRefreshRequest{
			RefreshToken: "valid-refresh-token",
		})
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var resp response.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success=true")
		}
	})

	t.Run("invalid or expired token", func(t *testing.T) {
		engine, handler, _, refreshUC, _, _, _ := setupAuthHandlerTest()
		engine.POST("/refresh", handler.Refresh)

		refreshUC.executeFn = func(ctx context.Context, req authtypes.TokenRefreshRequest, ipAddress, userAgent string) (*authtypes.TokenRefreshResponse, error) {
			return nil, domainerrors.ErrTokenExpired
		}

		body, _ := json.Marshal(authtypes.TokenRefreshRequest{
			RefreshToken: "expired-refresh-token",
		})
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("revoked token", func(t *testing.T) {
		engine, handler, _, refreshUC, _, _, _ := setupAuthHandlerTest()
		engine.POST("/refresh", handler.Refresh)

		refreshUC.executeFn = func(ctx context.Context, req authtypes.TokenRefreshRequest, ipAddress, userAgent string) (*authtypes.TokenRefreshResponse, error) {
			return nil, domainerrors.ErrTokenRevoked
		}

		body, _ := json.Marshal(authtypes.TokenRefreshRequest{
			RefreshToken: "revoked-refresh-token",
		})
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("inactive user during refresh", func(t *testing.T) {
		engine, handler, _, refreshUC, _, _, _ := setupAuthHandlerTest()
		engine.POST("/refresh", handler.Refresh)

		refreshUC.executeFn = func(ctx context.Context, req authtypes.TokenRefreshRequest, ipAddress, userAgent string) (*authtypes.TokenRefreshResponse, error) {
			return nil, domainerrors.ErrUserInactive
		}

		body, _ := json.Marshal(authtypes.TokenRefreshRequest{
			RefreshToken: "valid-token",
		})
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})
}

// withCaller registers a stand-in for the auth+tenant middleware that places the
// authenticated caller's identity on the request context via the ctx package.
func withCaller(tenantID, userID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		rc := ctx.WithTenantID(c.Request.Context(), tenantID.String())
		rc = ctx.WithUserID(rc, userID.String())
		c.Request = c.Request.WithContext(rc)
		c.Next()
	}
}

func TestAuthHandler_Logout_BindsCallerIdentity(t *testing.T) {
	t.Run("passes caller tenant and user from context, not body", func(t *testing.T) {
		engine, handler, _, _, logoutUC, _, _ := setupAuthHandlerTest()
		tenantID, userID := uuid.New(), uuid.New()
		engine.POST("/logout", withCaller(tenantID, userID), handler.Logout)

		called := false
		logoutUC.executeFn = func(ctx context.Context, tid, uid uuid.UUID, req authtypes.LogoutRequest) error {
			called = true
			if tid != tenantID || uid != userID {
				t.Errorf("expected caller %s/%s, got %s/%s", tenantID, userID, tid, uid)
			}
			return nil
		}

		body, _ := json.Marshal(authtypes.LogoutRequest{RefreshToken: "tok"})
		req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK || !called {
			t.Fatalf("expected 200 and usecase called, got %d called=%v", rec.Code, called)
		}
	})

	t.Run("token owned by another user yields 400", func(t *testing.T) {
		engine, handler, _, _, logoutUC, _, _ := setupAuthHandlerTest()
		engine.POST("/logout", withCaller(uuid.New(), uuid.New()), handler.Logout)
		logoutUC.executeFn = func(ctx context.Context, tid, uid uuid.UUID, req authtypes.LogoutRequest) error {
			return domainerrors.ErrInvalidToken
		}

		body, _ := json.Marshal(authtypes.LogoutRequest{RefreshToken: "someone-elses"})
		req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing caller identity yields 401 and usecase not called", func(t *testing.T) {
		engine, handler, _, _, logoutUC, _, _ := setupAuthHandlerTest()
		engine.POST("/logout", handler.Logout)
		logoutUC.executeFn = func(ctx context.Context, tid, uid uuid.UUID, req authtypes.LogoutRequest) error {
			t.Error("usecase must not be called without caller identity")
			return nil
		}

		body, _ := json.Marshal(authtypes.LogoutRequest{RefreshToken: "tok"})
		req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})
}

func TestAuthHandler_Refresh_ServerDerivedMetadata(t *testing.T) {
	engine, handler, _, refreshUC, _, _, _ := setupAuthHandlerTest()
	engine.POST("/refresh", handler.Refresh)

	var gotIP, gotUA string
	refreshUC.executeFn = func(ctx context.Context, req authtypes.TokenRefreshRequest, ipAddress, userAgent string) (*authtypes.TokenRefreshResponse, error) {
		gotIP, gotUA = ipAddress, userAgent
		return &authtypes.TokenRefreshResponse{}, nil
	}

	// A client-supplied ip_address/user_agent in the body must be ignored.
	body := []byte(`{"refresh_token":"t","ip_address":"6.6.6.6","user_agent":"spoofed"}`)
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "real-agent")
	req.RemoteAddr = "1.2.3.4:5555"
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotIP == "6.6.6.6" || gotUA == "spoofed" {
		t.Fatalf("client-supplied metadata leaked through: ip=%q ua=%q", gotIP, gotUA)
	}
	if gotUA != "real-agent" {
		t.Errorf("expected server-derived user agent, got %q", gotUA)
	}
}

func TestAuthHandler_Refresh_DoesNotLeakErrorText(t *testing.T) {
	engine, handler, _, refreshUC, _, _, _ := setupAuthHandlerTest()
	engine.POST("/refresh", handler.Refresh)
	refreshUC.executeFn = func(ctx context.Context, req authtypes.TokenRefreshRequest, ipAddress, userAgent string) (*authtypes.TokenRefreshResponse, error) {
		return nil, domainerrors.ErrTokenRevoked
	}
	body, _ := json.Marshal(authtypes.TokenRefreshRequest{RefreshToken: "t"})
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), domainerrors.ErrTokenRevoked.Error()) {
		t.Errorf("response leaks internal error text: %s", rec.Body.String())
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("successful logout", func(t *testing.T) {
		engine, handler, _, _, logoutUC, _, _ := setupAuthHandlerTest()
		engine.POST("/logout", withCaller(uuid.New(), uuid.New()), handler.Logout)

		logoutUC.executeFn = func(ctx context.Context, tenantID, userID uuid.UUID, req authtypes.LogoutRequest) error {
			if req.RefreshToken != "active-refresh-token" {
				t.Errorf("unexpected refresh token: %s", req.RefreshToken)
			}
			return nil
		}

		body, _ := json.Marshal(authtypes.LogoutRequest{
			RefreshToken: "active-refresh-token",
		})
		req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var resp response.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success=true")
		}
	})

	t.Run("invalid token error", func(t *testing.T) {
		engine, handler, _, _, logoutUC, _, _ := setupAuthHandlerTest()
		engine.POST("/logout", withCaller(uuid.New(), uuid.New()), handler.Logout)

		logoutUC.executeFn = func(ctx context.Context, tenantID, userID uuid.UUID, req authtypes.LogoutRequest) error {
			return domainerrors.ErrInvalidToken
		}

		body, _ := json.Marshal(authtypes.LogoutRequest{
			RefreshToken: "invalid-token",
		})
		req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestAuthHandler_ForgotPassword(t *testing.T) {
	t.Run("successful forgot password", func(t *testing.T) {
		engine, handler, _, _, _, forgotUC, _ := setupAuthHandlerTest()
		engine.POST("/forgot-password", handler.ForgotPassword)

		forgotUC.executeFn = func(ctx context.Context, req authtypes.ForgotPasswordRequest) error {
			if req.Email != "alice@example.com" {
				t.Errorf("unexpected email: %s", req.Email)
			}
			return nil
		}

		body, _ := json.Marshal(authtypes.ForgotPasswordRequest{
			Email: "alice@example.com",
		})
		req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp response.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success=true")
		}
	})

	t.Run("invalid json payload", func(t *testing.T) {
		engine, handler, _, _, _, _, _ := setupAuthHandlerTest()
		engine.POST("/forgot-password", handler.ForgotPassword)

		req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader([]byte("{invalid-json")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestAuthHandler_ResetPassword(t *testing.T) {
	t.Run("successful reset password", func(t *testing.T) {
		engine, handler, _, _, _, _, resetUC := setupAuthHandlerTest()
		engine.POST("/reset-password", handler.ResetPassword)

		resetUC.executeFn = func(ctx context.Context, req authtypes.ResetPasswordRequest) error {
			if req.Token != "valid-reset-token" || req.NewPassword != "newSecretPass123!" {
				t.Errorf("unexpected reset payload: %+v", req)
			}
			return nil
		}

		body, _ := json.Marshal(authtypes.ResetPasswordRequest{
			Token:       "valid-reset-token",
			NewPassword: "newSecretPass123!",
		})
		req := httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp response.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success=true")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		engine, handler, _, _, _, _, resetUC := setupAuthHandlerTest()
		engine.POST("/reset-password", handler.ResetPassword)

		resetUC.executeFn = func(ctx context.Context, req authtypes.ResetPasswordRequest) error {
			return domainerrors.ErrInvalidToken
		}

		body, _ := json.Marshal(authtypes.ResetPasswordRequest{
			Token:       "invalid-token",
			NewPassword: "newSecretPass123!",
		})
		req := httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("short password", func(t *testing.T) {
		engine, handler, _, _, _, _, resetUC := setupAuthHandlerTest()
		engine.POST("/reset-password", handler.ResetPassword)

		resetUC.executeFn = func(ctx context.Context, req authtypes.ResetPasswordRequest) error {
			return domainerrors.ErrInvalidPassword
		}

		body, _ := json.Marshal(authtypes.ResetPasswordRequest{
			Token:       "valid-token",
			NewPassword: "short",
		})
		req := httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}
