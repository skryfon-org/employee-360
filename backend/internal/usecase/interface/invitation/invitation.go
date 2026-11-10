package invitation

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
)

// InviteUserUseCase invites a user into the caller's tenant. The actor is
// resolved by the handler from the authenticated request. Only a super_admin
// actor may invite the admin role.
type InviteUserUseCase interface {
	Execute(ctx context.Context, actor emptypes.Actor, req invtypes.InviteUserRequest) (*entity.UserInvitation, error)
}

// AcceptInvitationUseCase consumes an invitation token; every invitee must set a password.
type AcceptInvitationUseCase interface {
	Execute(ctx context.Context, req invtypes.AcceptInvitationRequest) error
}

// ResendInvitationUseCase reissues the token of a pending invitation.
type ResendInvitationUseCase interface {
	Execute(ctx context.Context, actor emptypes.Actor, id uuid.UUID) (*entity.UserInvitation, error)
}

// RevokeInvitationUseCase cancels a pending invitation.
type RevokeInvitationUseCase interface {
	Execute(ctx context.Context, tenantID, actorID, id uuid.UUID) error
}

// ListInvitationsUseCase lists the caller's tenant invitations.
type ListInvitationsUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, q invtypes.ListInvitationsQuery) (*invtypes.ListInvitationsResult, error)
}

// ValidateInvitationUseCase verifies that an invitation token exists, is pending, not expired, and ready for acceptance.
type ValidateInvitationUseCase interface {
	Execute(ctx context.Context, token string) (*invtypes.ValidateInvitationResponse, error)
}
