package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skryfon-org/employee-360/backend/shared"
)

func newRequestIDEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestID())
	engine.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, GetRequestID(c))
	})
	return engine
}

func TestRequestID_GeneratesWhenAbsent(t *testing.T) {
	engine := newRequestIDEngine()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	respHeader := rec.Header().Get(shared.RequestIDHeader)
	if respHeader == "" {
		t.Fatal("expected a generated X-Request-ID header on the response")
	}
	if rec.Body.String() != respHeader {
		t.Errorf("expected context request id %q to match response header %q", rec.Body.String(), respHeader)
	}
}

func TestRequestID_PreservesClientSupplied(t *testing.T) {
	engine := newRequestIDEngine()

	const clientID = "client-supplied-id-123"
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(shared.RequestIDHeader, clientID)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	respHeader := rec.Header().Get(shared.RequestIDHeader)
	if respHeader != clientID {
		t.Errorf("expected client-supplied request id %q to be preserved, got %q", clientID, respHeader)
	}
	if rec.Body.String() != clientID {
		t.Errorf("expected context request id %q, got %q", clientID, rec.Body.String())
	}
}
