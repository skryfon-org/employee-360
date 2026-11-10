// Package invitation implements the onboarding-invitation usecases.
package invitation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

const (
	defaultInvitationExpiry = 7 * 24 * time.Hour
	roleAdmin               = "admin"
	roleSuperAdmin          = "super_admin"
	minPasswordLength       = 8
)

// requireIdentity rejects a nil tenant or actor ID. The handler resolves both
// from the authenticated request context; admin authorization is enforced by
// the route-level RequireRole middleware.
func requireIdentity(tenantID, actorID uuid.UUID) error {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	return nil
}

// requireCanGrantRole enforces that only a super_admin may grant the admin
// role through an invitation (mirrors the ChangeRole rule). Tenant admins may
// still invite employees.
func requireCanGrantRole(actor emptypes.Actor, roleName string) error {
	if roleName == roleAdmin && !actor.HasRole(roleSuperAdmin) {
		return domainerrors.ErrForbidden
	}
	return nil
}

func newPlainToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// AppURLs holds the public base URLs used to build invitation links. Admin and
// Employee are optional and fall back to Default when empty.
type AppURLs struct {
	Default  string
	Admin    string
	Employee string
}

// ForRole returns the base URL (no trailing slash) of the app the given role
// signs in to: admin/super_admin -> Admin, employee -> Employee, anything else
// (or an unset override) -> Default.
func (a AppURLs) ForRole(role string) string {
	u := ""
	switch role {
	case roleSuperAdmin, roleAdmin:
		u = a.Admin
	case "employee":
		u = a.Employee
	}
	if u == "" {
		u = a.Default
	}
	return strings.TrimRight(u, "/")
}

// acceptLink builds the invitation-accept link for the invited role's app.
func (a AppURLs) acceptLink(role, plainToken string) string {
	return a.ForRole(role) + "/accept-invitation?token=" + plainToken
}

// lookupUsableInvitation resolves a plaintext token to its invitation and the
// pending user it activates. An unknown token yields ErrInvalidToken; a known
// but unusable one yields the specific ErrInvitationExpired / Revoked /
// Accepted. The tenant is taken from the invitation row, never from input.
func lookupUsableInvitation(
	c context.Context,
	invRepo repository.UserInvitationRepository,
	userRepo repository.UserRepository,
	hashService service.HashService,
	token string,
	now time.Time,
) (*entity.UserInvitation, *entity.User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil, domainerrors.ErrInvalidToken
	}
	inv, err := invRepo.GetByTokenHash(c, hashService.HashToken(token))
	if err != nil || inv == nil {
		return nil, nil, domainerrors.ErrInvalidToken
	}
	switch {
	case inv.AcceptedAt != nil:
		return nil, nil, domainerrors.ErrInvitationAccepted
	case inv.RevokedAt != nil:
		return nil, nil, domainerrors.ErrInvitationRevoked
	case !inv.IsUsable(now):
		return nil, nil, domainerrors.ErrInvitationExpired
	}
	user, err := userRepo.GetByTenantAndEmail(c, inv.TenantID, inv.Email)
	if err != nil {
		if errors.Is(err, domainerrors.ErrNotFound) || errors.Is(err, domainerrors.ErrUserNotFound) {
			return nil, nil, domainerrors.ErrInvalidToken
		}
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, domainerrors.ErrInvalidToken
	}
	if user.IsActive {
		return nil, nil, domainerrors.ErrInvitationAccepted
	}
	return inv, user, nil
}
