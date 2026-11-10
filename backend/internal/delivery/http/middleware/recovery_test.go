package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestRecovery_ConvertsPanicToStandardizedEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuf bytes.Buffer
	log := zerolog.New(&logBuf)

	engine := gin.New()
	engine.Use(Recovery(log))
	engine.GET("/boom", func(c *gin.Context) {
		panic("something went wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()

	// The test itself must not crash even though the handler panics.
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"success":false`) {
		t.Errorf("expected a standardized error envelope, got: %s", body)
	}
	if !strings.Contains(body, "INTERNAL_ERROR") {
		t.Errorf("expected error code INTERNAL_ERROR, got: %s", body)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "something went wrong") {
		t.Errorf("expected panic value to be logged, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "panic recovered") {
		t.Errorf("expected 'panic recovered' log message, got: %s", logOutput)
	}
}

func TestRecovery_DoesNotInterfereWithNormalRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log := zerolog.New(&bytes.Buffer{})

	engine := gin.New()
	engine.Use(Recovery(log))
	engine.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "fine")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "fine" {
		t.Errorf("expected body 'fine', got %q", rec.Body.String())
	}
}
