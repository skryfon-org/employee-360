package ctx

import (
	"context"
	"testing"
)

func TestContextHelpers(t *testing.T) {
	bg := context.Background()

	t.Run("tenant ID", func(t *testing.T) {
		if _, ok := TenantIDFromContext(bg); ok {
			t.Fatal("expected no tenant ID in background context")
		}

		ctx := WithTenantID(bg, "tenant-123")
		val, ok := TenantIDFromContext(ctx)
		if !ok || val != "tenant-123" {
			t.Fatalf("expected tenant-123, got %s (ok=%v)", val, ok)
		}
	})

	t.Run("user ID", func(t *testing.T) {
		if _, ok := UserIDFromContext(bg); ok {
			t.Fatal("expected no user ID in background context")
		}

		ctx := WithUserID(bg, "user-456")
		val, ok := UserIDFromContext(ctx)
		if !ok || val != "user-456" {
			t.Fatalf("expected user-456, got %s (ok=%v)", val, ok)
		}
	})

	t.Run("roles", func(t *testing.T) {
		if _, ok := RolesFromContext(bg); ok {
			t.Fatal("expected no roles in background context")
		}

		ctx := WithRoles(bg, []string{"admin", "employee"})
		roles, ok := RolesFromContext(ctx)
		if !ok || len(roles) != 2 || roles[0] != "admin" || roles[1] != "employee" {
			t.Fatalf("unexpected roles: %v (ok=%v)", roles, ok)
		}
	})

	t.Run("client IP", func(t *testing.T) {
		if _, ok := ClientIPFromContext(bg); ok {
			t.Fatal("expected no client IP in background context")
		}

		ctx := WithClientIP(bg, "192.168.1.100")
		val, ok := ClientIPFromContext(ctx)
		if !ok || val != "192.168.1.100" {
			t.Fatalf("expected 192.168.1.100, got %s (ok=%v)", val, ok)
		}
	})

	t.Run("user agent", func(t *testing.T) {
		if _, ok := UserAgentFromContext(bg); ok {
			t.Fatal("expected no user agent in background context")
		}

		ctx := WithUserAgent(bg, "Mozilla/5.0")
		val, ok := UserAgentFromContext(ctx)
		if !ok || val != "Mozilla/5.0" {
			t.Fatalf("expected Mozilla/5.0, got %s (ok=%v)", val, ok)
		}
	})
}
