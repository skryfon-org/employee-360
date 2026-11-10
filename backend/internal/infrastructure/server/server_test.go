package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/skryfon-org/employee-360/backend/config"
)

// TestServer_StartAndShutdown is a lighter, isolated Start/Shutdown test
// (distinct from cmd/api/main_test.go's full-binary test): it binds an
// ephemeral port directly via Server, verifies the handler is reachable,
// shuts down gracefully, and confirms Start() returns nil and no longer
// accepts requests afterward.
func TestServer_StartAndShutdown(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	cfg := config.ServerConfig{
		Host: "127.0.0.1",
		Port: "18199",
	}
	srv := New(cfg, handler)

	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- srv.Start()
	}()

	url := "http://" + srv.Addr() + "/ping"
	var resp *http.Response
	var getErr error
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, getErr = http.Get(url)
		if getErr == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if getErr != nil {
		t.Fatalf("server never became reachable: %v", getErr)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from /ping, got %d", resp.StatusCode)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("expected graceful shutdown to succeed, got: %v", err)
	}

	select {
	case err := <-startErrCh:
		if err != nil {
			t.Errorf("expected Start() to return nil after graceful shutdown, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Start() to return after Shutdown")
	}

	if _, err := http.Get(url); err == nil {
		t.Error("expected requests to fail after shutdown, but one succeeded")
	}
}
