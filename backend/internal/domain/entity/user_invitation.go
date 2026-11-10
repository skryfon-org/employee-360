package entity

import (
	"time"

	"github.com/google/uuid"
)

// InvitationStatus is the derived lifecycle state of a UserInvitation.
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusRevoked  InvitationStatus = "revoked"
	InvitationStatusExpired  InvitationStatus = "expired"
)

// UserInvitation represents an admin-issued onboarding invitation. Only the
// hash of the invitation token is ever stored; the raw token is emailed.
type UserInvitation struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Email        string     `json:"email"`
	RoleID       uuid.UUID  `json:"role_id"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	PositionID   *uuid.UUID `json:"position_id,omitempty"`
	InvitedBy    uuid.UUID  `json:"invited_by"`
	TokenHash    string     `json:"-"`
	ExpiresAt    time.Time  `json:"expires_at"`
	AcceptedAt   *time.Time `json:"accepted_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CreatedBy    *uuid.UUID `json:"-"`
	UpdatedBy    *uuid.UUID `json:"-"`
	DeletedBy    *uuid.UUID `json:"-"`
	DeletedAt    *time.Time `json:"-"`
}

// IsPending reports whether the invitation has been neither accepted nor
// revoked. An expired-but-unconsumed invitation is still "pending" here so it
// can be resent or revoked; use IsUsable to check it can be accepted.
func (i *UserInvitation) IsPending() bool {
	return i.AcceptedAt == nil && i.RevokedAt == nil
}

// IsUsable reports whether the invitation can be accepted right now.
func (i *UserInvitation) IsUsable(now time.Time) bool {
	return i.IsPending() && now.Before(i.ExpiresAt)
}

// Status returns the derived lifecycle state at the given time.
func (i *UserInvitation) Status(now time.Time) InvitationStatus {
	switch {
	case i.AcceptedAt != nil:
		return InvitationStatusAccepted
	case i.RevokedAt != nil:
		return InvitationStatusRevoked
	case !now.Before(i.ExpiresAt):
		return InvitationStatusExpired
	default:
		return InvitationStatusPending
	}
}

// InvitationListItem is a read model for the admin listing: an invitation
// joined with its role name and inviting user (both resolved within the
// invitation's tenant).
type InvitationListItem struct {
	UserInvitation
	RoleName       string
	InvitedByFirst string
	InvitedByLast  string
	InvitedByEmail string
}
