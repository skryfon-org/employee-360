package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	healthtypes "github.com/skryfon-org/employee-360/backend/internal/types/health"

	"github.com/gin-gonic/gin"
)

// fakeHealthUseCase implements usecaseinterface.HealthUseCase for tests.
type fakeHealthUseCase struct {
	result healthtypes.HealthResult
}

func (f *fakeHealthUseCase) Execute(ctx context.Context) healthtypes.HealthResult {
	return f.result
}

func TestHealthHandler_Health(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fake := &fakeHealthUseCase{result: healthtypes.HealthResult{
		App:      "employee360",
		Database: "ok",
		Redis:    "disabled",
	}}
	handler := NewHealthHandler(fake)

	engine := gin.New()
	engine.GET("/api/v1/health", handler.Health)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Status   string `json:"status"`
			App      string `json:"app"`
			Database string `json:"database"`
			Redis    string `json:"redis"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response body: %v, body: %s", err, rec.Body.String())
	}

	if !body.Success {
		t.Error("expected success=true")
	}
	if body.Data.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", body.Data.Status)
	}
	if body.Data.App != "employee360" {
		t.Errorf("expected app 'employee360', got %q", body.Data.App)
	}
	if body.Data.Database != "ok" {
		t.Errorf("expected database 'ok', got %q", body.Data.Database)
	}
	if body.Data.Redis != "disabled" {
		t.Errorf("expected redis 'disabled', got %q", body.Data.Redis)
	}
}

func TestHealthHandler_Health_DatabaseUnreachable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fake := &fakeHealthUseCase{result: healthtypes.HealthResult{
		App:      "employee360",
		Database: "unreachable",
		Redis:    "unreachable",
	}}
	handler := NewHealthHandler(fake)

	engine := gin.New()
	engine.GET("/api/v1/health", handler.Health)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	// Health always responds 200, even when a dependency is unreachable.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"database":"unreachable"`) {
		t.Errorf("expected body to report database unreachable, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"redis":"unreachable"`) {
		t.Errorf("expected body to report redis unreachable, got: %s", rec.Body.String())
	}
}
