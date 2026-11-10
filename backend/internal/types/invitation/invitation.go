// Package invitation holds request DTOs for the onboarding-invitation usecases.
//
// None of these carry a tenant ID: the tenant is always resolved from the
// authenticated request context (or, for accept, from the token itself).
package invitation

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// InviteUserRequest is what an admin submits to invite a user.
type InviteUserRequest struct {
	Email        string     `json:"email"`
	RoleID       uuid.UUID  `json:"role_id"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	PositionID   *uuid.UUID `json:"position_id,omitempty"`
	FirstName    string     `json:"first_name,omitempty"`
	LastName     string     `json:"last_name,omitempty"`
}

// AcceptInvitationRequest is what an invitee submits to activate their account.
type AcceptInvitationRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// InvitationResponse is the API representation of an invitation (never includes the token).
type InvitationResponse struct {
	ID           uuid.UUID               `json:"id"`
	Email        string                  `json:"email"`
	RoleID       uuid.UUID               `json:"role_id"`
	DepartmentID *uuid.UUID              `json:"department_id,omitempty"`
	PositionID   *uuid.UUID              `json:"position_id,omitempty"`
	InvitedBy    uuid.UUID               `json:"invited_by"`
	Status       entity.InvitationStatus `json:"status"`
	ExpiresAt    time.Time               `json:"expires_at"`
	AcceptedAt   *time.Time              `json:"accepted_at,omitempty"`
	RevokedAt    *time.Time              `json:"revoked_at,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
}

// ToInvitationResponse maps an invitation entity to its API representation.
func ToInvitationResponse(i *entity.UserInvitation) InvitationResponse {
	return InvitationResponse{
		ID: i.ID, Email: i.Email, RoleID: i.RoleID, DepartmentID: i.DepartmentID,
		PositionID: i.PositionID, InvitedBy: i.InvitedBy, Status: i.Status(time.Now()),
		ExpiresAt: i.ExpiresAt, AcceptedAt: i.AcceptedAt, RevokedAt: i.RevokedAt, CreatedAt: i.CreatedAt,
	}
}

// Listing limits.
const (
	MaxSearchLen = 100
)

// ListInvitationsQuery is the validated input of the admin listing.
type ListInvitationsQuery struct {
	Page     int
	PageSize int
	Status   entity.InvitationStatus // empty = all
	Search   string
}

// ListInvitationsResult is one page of the listing plus pagination totals.
type ListInvitationsResult struct {
	Items      []*entity.InvitationListItem
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// InvitedByResponse identifies the admin who sent an invitation.
type InvitedByResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

// InvitationListItemResponse is one row of the admin invitation listing.
type InvitationListItemResponse struct {
	ID           uuid.UUID               `json:"id"`
	Email        string                  `json:"email"`
	Role         string                  `json:"role"`
	RoleID       uuid.UUID               `json:"role_id"`
	DepartmentID *uuid.UUID              `json:"department_id,omitempty"`
	PositionID   *uuid.UUID              `json:"position_id,omitempty"`
	InvitedBy    InvitedByResponse       `json:"invited_by"`
	InvitedOn    time.Time               `json:"invited_on"`
	Status       entity.InvitationStatus `json:"status"`
	ExpiresAt    time.Time               `json:"expires_at"`
	AcceptedAt   *time.Time              `json:"accepted_at,omitempty"`
	RevokedAt    *time.Time              `json:"revoked_at,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
}

// ToInvitationListItemResponse maps a listing row to its API representation.
func ToInvitationListItemResponse(i *entity.InvitationListItem) InvitationListItemResponse {
	name := strings.TrimSpace(i.InvitedByFirst + " " + i.InvitedByLast)
	return InvitationListItemResponse{
		ID: i.ID, Email: i.Email, Role: i.RoleName, RoleID: i.RoleID,
		DepartmentID: i.DepartmentID, PositionID: i.PositionID,
		InvitedBy: InvitedByResponse{ID: i.InvitedBy, Name: name, Email: i.InvitedByEmail},
		InvitedOn: i.CreatedAt, Status: i.Status(time.Now()), ExpiresAt: i.ExpiresAt,
		AcceptedAt: i.AcceptedAt, RevokedAt: i.RevokedAt, CreatedAt: i.CreatedAt,
	}
}

// ValidateInvitationResponse is returned when checking if an invitation token is valid before password entry.
type ValidateInvitationResponse struct {
	Email string `json:"email"`
	// Role is the invitee's role name (e.g. "admin", "employee"); clients use it
	// to send the invitee to the correct app.
	Role string `json:"role"`
}
