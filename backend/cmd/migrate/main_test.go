package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMigrate_DatabaseUnreachableFailsFast(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "up")
	cmd.Dir = "."
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
		t.Fatal("expected migrate to fail when database is unreachable, but it exited 0")
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

func TestFindMigrationsDir(t *testing.T) {
	dir := findMigrationsDir()
	if dir == "" {
		t.Fatal("expected findMigrationsDir to return a non-empty string")
	}
}
