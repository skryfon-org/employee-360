package container

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/skryfon-org/employee-360/backend/config"
)

// TestNew_ConstructsWithoutTouchingDB verifies the container can be built
// from a nil *gorm.DB: the database pinger wraps db lazily and is only
// dereferenced when Ping is actually called (e.g. by a request hitting the
// health route), not during construction.
func TestNew_ConstructsWithoutTouchingDB(t *testing.T) {
	cfg := &config.Config{
		CORS: config.CORSConfig{AllowedOrigins: []string{"*"}},
	}

	c, err := New(cfg, nil, nil, zerolog.Nop())
	if err != nil {
		t.Fatalf("expected New() to succeed, got: %v", err)
	}
	if c == nil {
		t.Fatal("expected a non-nil Container")
	}
	if c.Health == nil || c.Health.Handler == nil {
		t.Fatal("expected Health sub-container and its handler to be wired")
	}
	if c.Auth == nil || c.Auth.Handler == nil {
		t.Fatal("expected Auth sub-container and its handler to be wired")
	}
	if c.Cache == nil {
		t.Fatal("expected a no-op Cache when none is supplied")
	}
	if c.Department == nil || c.Department.Handler == nil {
		t.Fatal("expected Department sub-container and its handler to be wired")
	}
	if c.Position == nil || c.Position.Handler == nil {
		t.Fatal("expected Position sub-container and its handler to be wired")
	}
	if c.AuditLog == nil || c.AuditLog.Handler == nil {
		t.Fatal("expected AuditLog sub-container and its handler to be wired")
	}
	if c.Holiday == nil || c.Holiday.Handler == nil {
		t.Fatal("expected Holiday sub-container and its handler to be wired")
	}
}
