//go:build integration

// Package pgtest gives integration tests a dedicated PostgreSQL database so
// they never read or write the dev database. The test database lives on the
// same server as the configured one and is named "<DATABASE_NAME>_test"; it is
// created and migrated on first use and reused afterwards.
package pgtest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

const testSuffix = "_test"

var (
	prepareMu sync.Mutex
	prepared  = map[string]bool{}
)

func inCI() bool {
	return os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
}

// skipOrFatal skips the test locally but fails it in CI, where a missing
// database must not silently turn into a green run.
func skipOrFatal(t *testing.T, format string, args ...any) {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	if inCI() {
		t.Fatal(msg)
	}
	t.Skip("skipping live database test: " + msg)
}

// allowTestBaseEnv opts in to using a configured database name that already
// ends in "_test" as the test database as-is (instead of refusing it).
const allowTestBaseEnv = "PGTEST_ALLOW_TEST_BASE"

// TestDBName returns the test database name derived from the configured one.
// It always appends the suffix, so the result can never equal the configured
// database.
func TestDBName(base string) string {
	return base + testSuffix
}

// resolveDBName maps the configured database name to the test database name.
// A configured name that already ends in "_test" is refused unless explicitly
// opted in, since it is more likely a mis-pointed environment than a deliberate
// choice.
func resolveDBName(t *testing.T, base string) string {
	t.Helper()
	if strings.HasSuffix(base, testSuffix) {
		if os.Getenv(allowTestBaseEnv) == "true" {
			return base
		}
		t.Fatalf("configured database %q already ends in %q; refusing to guess. Set %s=true to use it as the test database", base, testSuffix, allowTestBaseEnv)
	}
	return TestDBName(base)
}

// load loads the app config and returns it pointed at the test database,
// together with the originally configured database name.
func load(t *testing.T, searchPaths ...string) (*config.Config, string) {
	t.Helper()
	cfg, err := config.Load(searchPaths...)
	if err != nil {
		skipOrFatal(t, "failed to load config: %v", err)
	}
	base := cfg.Database.DBName
	cfg.Database.DBName = resolveDBName(t, base)
	// ConnURL would be re-parsed over DBName by consumers; the discrete fields
	// are authoritative from here on.
	cfg.Database.ConnURL = ""
	return cfg, base
}

// Config loads the app config (searching the given extra paths for .env) and
// points its database settings at the test database. It does not create it;
// use Prepare or Open for that.
func Config(t *testing.T, searchPaths ...string) *config.Config {
	t.Helper()
	cfg, _ := load(t, searchPaths...)
	return cfg
}

// Env returns environment overrides that point a child process (which loads
// config itself) at the test database. Every alias config accepts for the
// database name is overridden so none can shadow it.
func Env(dbName string) []string {
	return []string{
		"DATABASE_URL=",
		"DATABASE_NAME=" + dbName,
		"DATABASE_DBNAME=" + dbName,
		"DB_NAME=" + dbName,
		"POSTGRES_DB=" + dbName,
	}
}

// Prepare loads config once, ensures the test database exists and is migrated,
// and returns the config pointed at it. Use it when only the config is needed
// (e.g. to launch a child process); use Open when a connection is needed too.
func Prepare(t *testing.T, searchPaths ...string) *config.Config {
	t.Helper()
	cfg, base := load(t, searchPaths...)
	guard(t, cfg.Database.DBName, base)
	ensureDatabase(t, cfg)
	return cfg
}

// guard refuses to proceed unless the target is a "_test" database distinct
// from the one the app is configured with.
func guard(t *testing.T, target, base string) {
	t.Helper()
	if !strings.HasSuffix(target, testSuffix) {
		t.Fatalf("refusing to run integration tests against non-test database %q", target)
	}
	if target == base && os.Getenv(allowTestBaseEnv) != "true" {
		t.Fatalf("refusing to run integration tests against the configured database %q", target)
	}
}

// Open returns a connection to the migrated test database. The connection is
// closed when the test ends. searchPaths are extra directories in which to
// look for .env, relative to the calling test's package.
func Open(t *testing.T, searchPaths ...string) *gorm.DB {
	t.Helper()
	cfg := Prepare(t, searchPaths...)

	db, err := database.Connect(cfg.Database)
	if err != nil {
		skipOrFatal(t, "PostgreSQL unreachable: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				t.Logf("pgtest: failed to close test database connection: %v", err)
			}
		}
	})
	return db
}

// isBenignCreateRace reports whether err from CREATE DATABASE only means a
// concurrent package process created the same database first.
func isBenignCreateRace(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42P04": // duplicate_database
			return true
		case "23505": // unique_violation on pg_database_datname_index
			return strings.Contains(pgErr.ConstraintName, "pg_database_datname_index") ||
				strings.Contains(pgErr.Message, "pg_database_datname_index")
		}
		return strings.Contains(pgErr.Message, "tuple concurrently updated")
	}
	msg := err.Error()
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "pg_database_datname_index") ||
		strings.Contains(msg, "tuple concurrently updated")
}

// ensureDatabase creates and migrates the test database once per process.
func ensureDatabase(t *testing.T, cfg *config.Config) {
	t.Helper()
	dbCfg := cfg.Database

	prepareMu.Lock()
	defer prepareMu.Unlock()
	if prepared[dbCfg.URL()] {
		return
	}

	// Connect through the maintenance database, never the dev one.
	adminCfg := dbCfg
	adminCfg.DBName = "postgres"
	admin, err := database.Connect(adminCfg)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// The server answered, so it is reachable: access to the
			// maintenance database was refused.
			skipOrFatal(t, "cannot connect to the %q maintenance database (needs access to it to create %q): %v", adminCfg.DBName, dbCfg.DBName, err)
		}
		skipOrFatal(t, "PostgreSQL unreachable: %v", err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	defer func() {
		if err := adminSQL.Close(); err != nil {
			t.Logf("pgtest: failed to close admin connection: %v", err)
		}
	}()

	var exists bool
	if err := admin.Raw("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)", dbCfg.DBName).Scan(&exists).Error; err != nil {
		t.Fatalf("failed to check for test database: %v", err)
	}
	if !exists {
		// Name is the configured one plus a fixed suffix; quote it anyway.
		if _, err := adminSQL.Exec(`CREATE DATABASE "` + dbCfg.DBName + `"`); err != nil {
			// Another package process may have created it between the check and here.
			if !isBenignCreateRace(err) {
				skipOrFatal(t, "cannot create test database %q (role needs the CREATEDB privilege): %v", dbCfg.DBName, err)
			}
		}
	}

	m, err := migrate.New("file://"+migrationsDir(t), dbCfg.URL())
	if err != nil {
		t.Fatalf("failed to initialize migrator: %v", err)
	}
	defer func() { _, _ = m.Close() }()
	// golang-migrate holds a Postgres advisory lock during Up(), so concurrent
	// packages applying the same migrations are safe.
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("failed to apply migrations: %v", err)
	}
	prepared[dbCfg.URL()] = true
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller for migrationsDir")
	}
	// this file: backend/internal/testutil/pgtest/pgtest.go
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatalf("failed to resolve migrations dir: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected migrations directory at %s: %v", dir, err)
	}
	return dir
}
