package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestLogger_LogsRequestFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuf bytes.Buffer
	log := zerolog.New(&logBuf)

	engine := gin.New()
	engine.Use(RequestID(), Logger(log))
	engine.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var logLine map[string]interface{}
	if err := json.Unmarshal(logBuf.Bytes(), &logLine); err != nil {
		t.Fatalf("failed to unmarshal log line: %v, raw: %s", err, logBuf.String())
	}

	for _, field := range []string{"method", "path", "status", "latency", "request_id"} {
		if _, ok := logLine[field]; !ok {
			t.Errorf("expected log line to contain field %q, got: %s", field, logBuf.String())
		}
	}

	if logLine["method"] != "GET" {
		t.Errorf("expected method GET, got %v", logLine["method"])
	}
	if logLine["path"] != "/ping" {
		t.Errorf("expected path /ping, got %v", logLine["path"])
	}
	if status, ok := logLine["status"].(float64); !ok || int(status) != http.StatusOK {
		t.Errorf("expected status 200, got %v", logLine["status"])
	}
	if logLine["request_id"] == "" {
		t.Error("expected a non-empty request_id in the log line")
	}
}
