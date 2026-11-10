package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// UserInvitationRepository defines the data access methods for user invitations.
//
// Like UserRepository, tenant-scoped methods take an explicit tenantID that
// the calling usecase obtained from the verified request context; it is never
// client-supplied. GetByTokenHash is the sole unscoped lookup: the accept flow
// is unauthenticated and resolves the tenant from the token itself.
type UserInvitationRepository interface {
	Create(ctx context.Context, invitation *entity.UserInvitation) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.UserInvitation, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*entity.UserInvitation, error)
	// UpdateToken replaces the token hash and expiry (resend). It is a
	// conditional update (accepted_at IS NULL AND revoked_at IS NULL) and MUST
	// return domainerrors.ErrInvitationNotPending when 0 rows were updated.
	UpdateToken(ctx context.Context, tenantID, id, actorID uuid.UUID, tokenHash string, expiresAt time.Time) error
	// MarkAccepted is a conditional update (accepted_at IS NULL AND
	// revoked_at IS NULL) and MUST return domainerrors.ErrInvitationNotPending
	// when 0 rows were updated (the accept usecase maps it to ErrInvalidToken).
	MarkAccepted(ctx context.Context, tenantID, id uuid.UUID, at time.Time) error
	// MarkRevoked is a conditional update (accepted_at IS NULL AND
	// revoked_at IS NULL) and MUST return domainerrors.ErrInvitationNotPending
	// when 0 rows were updated.
	MarkRevoked(ctx context.Context, tenantID, id, actorID uuid.UUID, at time.Time) error
	// HasAcceptedForEmail reports whether the tenant has any accepted
	// (non-deleted) invitation for the email.
	HasAcceptedForEmail(ctx context.Context, tenantID uuid.UUID, email string) (bool, error)
	// RevokeUnacceptedForEmail revokes every still-unaccepted, unrevoked
	// invitation (pending or expired) for the email in the tenant and returns
	// how many rows were revoked. Used when re-inviting an inactive user.
	RevokeUnacceptedForEmail(ctx context.Context, tenantID uuid.UUID, email string, actorID uuid.UUID, at time.Time) (int64, error)
	// List returns one page of the tenant's invitations (joined with role name
	// and inviter) plus the total number of rows matching the filter. Ordering
	// is created_at DESC, id DESC.
	List(ctx context.Context, tenantID uuid.UUID, filter InvitationListFilter) ([]*entity.InvitationListItem, int64, error)
}

// InvitationListFilter narrows and pages an invitation listing.
type InvitationListFilter struct {
	// Status, when non-empty, keeps only invitations in that derived state
	// (evaluated against Now; expired = pending-but-past expires_at).
	Status entity.InvitationStatus
	// Search is a case-insensitive substring match on the invitee email. The
	// repository escapes LIKE wildcards.
	Search string
	Now    time.Time
	Limit  int
	Offset int
}
