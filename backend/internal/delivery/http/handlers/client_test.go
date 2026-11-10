package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetClientInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("extracts IP and user agent correctly", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, err := http.NewRequest(http.MethodPost, "/test", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("User-Agent", "TestBrowser/2.0")
		req.RemoteAddr = "192.0.2.1:1234"
		c.Request = req

		info := GetClientInfo(c)
		if info.IPAddress != "192.0.2.1" {
			t.Errorf("expected IP 192.0.2.1, got %q", info.IPAddress)
		}
		if info.UserAgent != "TestBrowser/2.0" {
			t.Errorf("expected User-Agent 'TestBrowser/2.0', got %q", info.UserAgent)
		}
	})

	t.Run("handles nil context gracefully", func(t *testing.T) {
		info := GetClientInfo(nil)
		if info.IPAddress != "" {
			t.Errorf("expected empty IP, got %q", info.IPAddress)
		}
		if info.UserAgent != "" {
			t.Errorf("expected empty User-Agent, got %q", info.UserAgent)
		}
	})

	t.Run("handles nil request gracefully", func(t *testing.T) {
		c := &gin.Context{}
		info := GetClientInfo(c)
		if info.UserAgent != "" {
			t.Errorf("expected empty User-Agent, got %q", info.UserAgent)
		}
	})
}
