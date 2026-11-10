package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// EmergencyContactRepository persists emergency_contacts. Tenant-scoped,
// soft-delete aware. At most one live primary contact per user is guaranteed by
// a partial unique index; callers clear the old primary first.
type EmergencyContactRepository interface {
	ListByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.EmergencyContact, error)
	GetByID(ctx context.Context, tenantID, userID, id uuid.UUID) (*entity.EmergencyContact, error)
	CountByUser(ctx context.Context, tenantID, userID uuid.UUID) (int64, error)
	Create(ctx context.Context, actorID uuid.UUID, contact *entity.EmergencyContact) error
	Update(ctx context.Context, tenantID, actorID uuid.UUID, contact *entity.EmergencyContact) error
	// Delete soft-deletes the contact.
	Delete(ctx context.Context, tenantID, userID, id, actorID uuid.UUID) error
	// ClearPrimary unsets is_primary on the user's live contacts other than exceptID.
	ClearPrimary(ctx context.Context, tenantID, userID, exceptID, actorID uuid.UUID) error
}
