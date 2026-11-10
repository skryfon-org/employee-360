package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildAPIBinary compiles cmd/api into a temp binary. Running a real
// binary (rather than "go run .") makes SIGTERM delivery deterministic,
// which the graceful-shutdown assertions below depend on.
func buildAPIBinary(t *testing.T) string {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "api-test-bin")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = "."

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build api binary: %v, stderr: %s", err, stderr.String())
	}
	return binPath
}

// TestAPI_DatabaseUnreachableFailsFast verifies the api binary still
// fails fast (before ever binding a listener) when the database is
// unreachable, exactly as it did prior to the server/graceful-shutdown
// work — the fail-fast check runs before the server starts.
func TestAPI_DatabaseUnreachableFailsFast(t *testing.T) {
	binPath := buildAPIBinary(t)

	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"APP_ENV=test",
		"DATABASE_HOST=127.0.0.1",
		"DATABASE_PORT=59999",
		"DATABASE_CONN_TIMEOUT=500ms",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected api to fail when database is unreachable, but it exited 0")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got: %v", err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "database unreachable") {
		t.Errorf("expected stderr to contain 'database unreachable', got: %q", stderrStr)
	}
}
