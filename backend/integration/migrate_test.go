//go:build integration

package integration

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	"github.com/skryfon-org/employee-360/backend/internal/testutil/pgtest"
)

// migrateDir is cmd/migrate's location relative to this package
// (integration/), used as the working directory for "go run ."
// invocations below.
const migrateDir = "../cmd/migrate"

// realMigrationsDir is the real backend/migrations directory, relative to
// this package (integration/) -- the same directory findMigrationsDir() in
// cmd/migrate/main.go resolves to by default when a "go run ." invocation's
// working directory is migrateDir (backend/cmd/migrate) and no -dir flag is
// given: its candidate list tries "migrations" and "backend/migrations"
// (relative to backend/cmd/migrate, neither exists) before "../migrations",
// which resolves to backend/migrations.
const realMigrationsDir = "../migrations"

// countRealUpMigrations counts the real .up.sql files in realMigrationsDir
// and returns that count along with the highest sequence number found in
// their filenames (the version golang-migrate will report once all of them
// are applied). Counted dynamically so this test doesn't go stale as more
// migrations are added.
func countRealUpMigrations(t *testing.T) (count int, maxVersion int) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(realMigrationsDir, "*.up.sql"))
	if err != nil {
		t.Fatalf("failed to glob real migrations directory: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected at least one real migration in %s, found none", realMigrationsDir)
	}

	for _, m := range matches {
		base := filepath.Base(m)
		seqStr, _, ok := strings.Cut(base, "_")
		if !ok {
			t.Fatalf("migration filename %q does not match NNNNNN_description.up.sql convention", base)
		}
		seq, err := strconv.Atoi(seqStr)
		if err != nil {
			t.Fatalf("migration filename %q has a non-numeric sequence prefix: %v", base, err)
		}
		if seq > maxVersion {
			maxVersion = seq
		}
	}

	return len(matches), maxVersion
}

func inCI() bool {
	return os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
}

// scratchDBEnv creates a uniquely named scratch database on the configured
// PostgreSQL server and returns an environment (os.Environ() plus overrides)
// that points cmd/migrate child processes at it. The scratch database is
// dropped on test cleanup, so these tests never touch the dev database's
// golang-migrate version table or schema.
//
// Skips when PostgreSQL is unreachable or the role cannot CREATE DATABASE
// outside CI; both are fatal in CI.
func scratchDBEnv(t *testing.T) []string {
	t.Helper()
	// Only used to reach the server; scratch databases are created next to the
	// (guarded, migrated) test database, never the dev one.
	cfg := pgtest.Prepare(t)
	admin, err := database.Connect(cfg.Database)
	if err != nil {
		if inCI() {
			t.Fatalf("PostgreSQL must be reachable in CI: %v", err)
		}
		t.Skipf("skipping live database test (PostgreSQL unreachable: %v)", err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatalf("failed to generate random db name: %v", err)
	}
	// Fixed prefix + lowercase hex: identifier-safe, no user input.
	name := "employee360_migtest_" + hex.EncodeToString(suffix)

	if _, err := adminSQL.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
		_ = adminSQL.Close()
		if inCI() {
			t.Fatalf("role must be able to CREATE DATABASE in CI: %v", err)
		}
		t.Skipf("skipping live database test (cannot CREATE DATABASE: %v)", err)
	}

	t.Cleanup(func() {
		defer adminSQL.Close()
		if _, err := adminSQL.Exec(`DROP DATABASE IF EXISTS "` + name + `" WITH (FORCE)`); err != nil {
			t.Errorf("failed to drop scratch database %s: %v", name, err)
		}
	})

	// Override every alias config.bindEnvAliases accepts for the DB name so
	// no pre-existing variable can shadow the scratch database.
	return append(os.Environ(), pgtest.Env(name)...)
}

func TestMigrate_LiveDatabaseZeroMigrationsClean(t *testing.T) {
	env := scratchDBEnv(t)

	// Deliberately use an explicit empty tempDir here (via -dir), not the
	// default auto-detected directory: findMigrationsDir() now resolves to
	// the real backend/migrations directory, which is no longer empty since
	// Cycle 2 added real migrations. Pointing this test at the real
	// migrations directory would apply the full real schema instead of
	// exercising the "zero migrations" behavior this test is named for, and
	// would conflict with TestMigrate_LiveDatabaseRealMigrationsApplyAndRevert
	// below (which owns exercising the real directory end-to-end). Using an
	// isolated empty dir keeps this test's original intent while leaving the
	// real schema state untouched.
	emptyDir := t.TempDir()

	// Test "up" with zero migrations (AC-4 & AC-5)
	cmdUp := exec.Command("go", "run", ".", "-dir", emptyDir, "up")
	cmdUp.Dir = migrateDir
	cmdUp.Env = env
	var stdoutUp, stderrUp bytes.Buffer
	cmdUp.Stdout = &stdoutUp
	cmdUp.Stderr = &stderrUp

	if err := cmdUp.Run(); err != nil {
		t.Fatalf("expected migrate up to succeed with zero migrations, got err: %v, stderr: %s", err, stderrUp.String())
	}
	if !strings.Contains(stdoutUp.String(), "migrations applied successfully") {
		t.Errorf("expected stdout to contain 'migrations applied successfully', got: %q", stdoutUp.String())
	}

	// Test "status" with zero migrations
	cmdStatus := exec.Command("go", "run", ".", "-dir", emptyDir, "status")
	cmdStatus.Dir = migrateDir
	cmdStatus.Env = env
	var stdoutStatus, stderrStatus bytes.Buffer
	cmdStatus.Stdout = &stdoutStatus
	cmdStatus.Stderr = &stderrStatus

	if err := cmdStatus.Run(); err != nil {
		t.Fatalf("expected migrate status to succeed, got err: %v, stderr: %s", err, stderrStatus.String())
	}
	if !strings.Contains(stdoutStatus.String(), "no migrations applied yet") {
		t.Errorf("expected stdout to contain 'no migrations applied yet', got: %q", stdoutStatus.String())
	}
}

func TestMigrate_LiveDatabaseApplyAndRevert(t *testing.T) {
	env := scratchDBEnv(t)

	tempDir := t.TempDir()
	upContent := "CREATE TABLE IF NOT EXISTS _smoke_test_table (id serial primary key, name text);\n"
	downContent := "DROP TABLE IF EXISTS _smoke_test_table;\n"

	if err := os.WriteFile(filepath.Join(tempDir, "000001_smoke_test.up.sql"), []byte(upContent), 0644); err != nil {
		t.Fatalf("failed to write up migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "000001_smoke_test.down.sql"), []byte(downContent), 0644); err != nil {
		t.Fatalf("failed to write down migration: %v", err)
	}

	// 1. Run UP
	cmdUp := exec.Command("go", "run", ".", "-dir", tempDir, "up")
	cmdUp.Dir = migrateDir
	cmdUp.Env = env
	var stdoutUp, stderrUp bytes.Buffer
	cmdUp.Stdout = &stdoutUp
	cmdUp.Stderr = &stderrUp

	if err := cmdUp.Run(); err != nil {
		t.Fatalf("expected migrate up to succeed, got: %v, stderr: %s", err, stderrUp.String())
	}
	if !strings.Contains(stdoutUp.String(), "migrations applied successfully") {
		t.Errorf("expected stdout to contain 'migrations applied successfully', got: %q", stdoutUp.String())
	}

	// 2. Check STATUS
	cmdStatus := exec.Command("go", "run", ".", "-dir", tempDir, "status")
	cmdStatus.Dir = migrateDir
	cmdStatus.Env = env
	var stdoutStatus, stderrStatus bytes.Buffer
	cmdStatus.Stdout = &stdoutStatus
	cmdStatus.Stderr = &stderrStatus

	if err := cmdStatus.Run(); err != nil {
		t.Fatalf("expected migrate status to succeed, got: %v, stderr: %s", err, stderrStatus.String())
	}
	if !strings.Contains(stdoutStatus.String(), "current version: 1") {
		t.Errorf("expected status to contain 'current version: 1', got: %q", stdoutStatus.String())
	}

	// 3. Run DOWN
	cmdDown := exec.Command("go", "run", ".", "-dir", tempDir, "down", "1")
	cmdDown.Dir = migrateDir
	cmdDown.Env = env
	var stdoutDown, stderrDown bytes.Buffer
	cmdDown.Stdout = &stdoutDown
	cmdDown.Stderr = &stderrDown

	if err := cmdDown.Run(); err != nil {
		t.Fatalf("expected migrate down to succeed, got: %v, stderr: %s", err, stderrDown.String())
	}
	if !strings.Contains(stdoutDown.String(), "rolled back 1 migration(s)") {
		t.Errorf("expected stdout to contain 'rolled back 1 migration(s)', got: %q", stdoutDown.String())
	}
}

// TestMigrate_LiveDatabaseRealMigrationsApplyAndRevert is the one test in
// this package that exercises the real backend/migrations directory
// end-to-end: apply every real migration, then revert every real
// down-script. Every other test in this file operates against a synthetic,
// isolated tempDir and never touches the real schema, so this is the only
// test that mutates real migration state -- it is written to be safe to run
// repeatedly (it always finishes by rolling all the way back to version 0)
// and does not depend on execution order relative to the other tests in
// this file.
//
// This closes the gap flagged in an earlier review: `make migrate`
// (up) against the real directory was smoke-tested in CI, but `make
// migrate-down` against the real down-scripts never actually ran anywhere.
func TestMigrate_LiveDatabaseRealMigrationsApplyAndRevert(t *testing.T) {
	env := scratchDBEnv(t)

	count, maxVersion := countRealUpMigrations(t)

	// 1. Run UP against the real migrations directory. No -dir flag: this
	// relies on findMigrationsDir()'s default auto-detection resolving to
	// the real backend/migrations directory (see realMigrationsDir above),
	// exactly as `make migrate` does in production.
	cmdUp := exec.Command("go", "run", ".", "up")
	cmdUp.Dir = migrateDir
	cmdUp.Env = env
	var stdoutUp, stderrUp bytes.Buffer
	cmdUp.Stdout = &stdoutUp
	cmdUp.Stderr = &stderrUp

	if err := cmdUp.Run(); err != nil {
		t.Fatalf("expected migrate up to succeed against the real migrations directory, got: %v, stderr: %s", err, stderrUp.String())
	}
	if !strings.Contains(stdoutUp.String(), "migrations applied successfully") {
		t.Errorf("expected stdout to contain 'migrations applied successfully', got: %q", stdoutUp.String())
	}

	// 2. Check STATUS reports the final real version (dynamically computed,
	// not hardcoded, so this doesn't go stale as migrations are added).
	cmdStatus := exec.Command("go", "run", ".", "status")
	cmdStatus.Dir = migrateDir
	cmdStatus.Env = env
	var stdoutStatus, stderrStatus bytes.Buffer
	cmdStatus.Stdout = &stdoutStatus
	cmdStatus.Stderr = &stderrStatus

	if err := cmdStatus.Run(); err != nil {
		t.Fatalf("expected migrate status to succeed, got: %v, stderr: %s", err, stderrStatus.String())
	}
	wantStatus := fmt.Sprintf("current version: %d", maxVersion)
	if !strings.Contains(stdoutStatus.String(), wantStatus) {
		t.Errorf("expected status to contain %q, got: %q", wantStatus, stdoutStatus.String())
	}
	if strings.Contains(stdoutStatus.String(), "DIRTY") {
		t.Fatalf("database left in a dirty state after applying all real migrations: %q", stdoutStatus.String())
	}

	// 3. Run DOWN for the full real count, reverting every real down-script
	// in sequence -- the part of AC-3 that had no automated coverage.
	cmdDown := exec.Command("go", "run", ".", "down", strconv.Itoa(count))
	cmdDown.Dir = migrateDir
	cmdDown.Env = env
	var stdoutDown, stderrDown bytes.Buffer
	cmdDown.Stdout = &stdoutDown
	cmdDown.Stderr = &stderrDown

	if err := cmdDown.Run(); err != nil {
		t.Fatalf("expected migrate down to succeed against the real migrations directory, got: %v, stderr: %s", err, stderrDown.String())
	}
	wantDown := fmt.Sprintf("rolled back %d migration(s)", count)
	if !strings.Contains(stdoutDown.String(), wantDown) {
		t.Errorf("expected stdout to contain %q, got: %q", wantDown, stdoutDown.String())
	}

	// 4. Confirm we're back to a clean slate (version 0 / no migrations
	// applied), so this test is safe to re-run and leaves no state behind
	// for other tests in this package.
	cmdFinalStatus := exec.Command("go", "run", ".", "status")
	cmdFinalStatus.Dir = migrateDir
	cmdFinalStatus.Env = env
	var stdoutFinalStatus, stderrFinalStatus bytes.Buffer
	cmdFinalStatus.Stdout = &stdoutFinalStatus
	cmdFinalStatus.Stderr = &stderrFinalStatus

	if err := cmdFinalStatus.Run(); err != nil {
		t.Fatalf("expected migrate status to succeed after full revert, got: %v, stderr: %s", err, stderrFinalStatus.String())
	}
	if !strings.Contains(stdoutFinalStatus.String(), "no migrations applied yet") {
		t.Errorf("expected stdout to contain 'no migrations applied yet' after full revert, got: %q", stdoutFinalStatus.String())
	}
}
