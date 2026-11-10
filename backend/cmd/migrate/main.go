package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

func main() {
	migrationsDirFlag := flag.String("dir", "", "migrations directory (default: auto-detect)")
	flag.Parse()

	dir := *migrationsDirFlag
	if dir == "" {
		dir = findMigrationsDir()
	}

	args := flag.Args()
	cmd := "up"
	if len(args) > 0 {
		cmd = strings.ToLower(args[0])
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("failed to load configuration: " + err.Error())
	}

	// Verify database connectivity (fail fast with "database unreachable" if down)
	db, err := database.Connect(cfg.Database)
	if err != nil {
		database.FailFast(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	dbURL := cfg.Database.URL()

	switch cmd {
	case "up":
		runUp(dbURL, dir)
	case "down":
		n := 1
		if len(args) > 1 {
			var err error
			n, err = strconv.Atoi(args[1])
			if err != nil {
				fatal("down: invalid step count: " + args[1])
			}
		}
		runDown(dbURL, dir, n)
	case "status":
		runStatus(dbURL, dir)
	case "force":
		if len(args) < 2 {
			fatal("usage: migrate force <version>")
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			fatal("force: invalid version: " + args[1])
		}
		runForce(dbURL, dir, v)
	default:
		fatal("unknown command: " + cmd)
	}
}

func newMigrate(dbURL, dir string) (*migrate.Migrate, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	return migrate.New("file://"+absDir, dbURL)
}

func runUp(dbURL, dir string) {
	m, err := newMigrate(dbURL, dir)
	if err != nil {
		fatal("up: " + err.Error())
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !isNoChangeOrEmpty(err) {
		fatal("up: " + err.Error())
	}
	fmt.Println("migrations applied successfully")
}

func runDown(dbURL, dir string, steps int) {
	m, err := newMigrate(dbURL, dir)
	if err != nil {
		fatal("down: " + err.Error())
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Steps(-steps); err != nil && !isNoChangeOrEmpty(err) {
		fatal("down: " + err.Error())
	}
	fmt.Printf("rolled back %d migration(s)\n", steps)
}

func runForce(dbURL, dir string, version int) {
	m, err := newMigrate(dbURL, dir)
	if err != nil {
		fatal("force: " + err.Error())
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Force(version); err != nil {
		fatal("force: " + err.Error())
	}
	fmt.Printf("forced version to %d\n", version)
}

func runStatus(dbURL, dir string) {
	m, err := newMigrate(dbURL, dir)
	if err != nil {
		fatal("status: " + err.Error())
	}
	defer func() { _, _ = m.Close() }()

	version, dirty, err := m.Version()
	if err != nil {
		if isNoChangeOrEmpty(err) {
			fmt.Println("no migrations applied yet")
			return
		}
		fatal("status: " + err.Error())
	}
	dirtyStr := ""
	if dirty {
		dirtyStr = " (DIRTY)"
	}
	fmt.Printf("current version: %d%s\n", version, dirtyStr)
}

func isNoChangeOrEmpty(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, migrate.ErrNoChange) || errors.Is(err, migrate.ErrNilVersion) || errors.Is(err, os.ErrNotExist) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "file does not exist") || strings.Contains(msg, "no change")
}

func findMigrationsDir() string {
	candidates := []string{
		"migrations",
		"backend/migrations",
		"../migrations",
		"../../migrations",
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}

	return "migrations"
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "migrate:", msg)
	os.Exit(1)
}
