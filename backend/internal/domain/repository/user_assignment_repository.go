package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// UserAssignmentRepository persists the department/position history. Tenant
// scoped and soft-delete aware.
type UserAssignmentRepository interface {
	// GetOpen returns the user's current (effective_to IS NULL) stint, or
	// (nil, nil) when there is none.
	GetOpen(ctx context.Context, tenantID, userID uuid.UUID) (*entity.UserAssignment, error)
	// CloseOpen sets effective_to on the open stint (no-op when none).
	CloseOpen(ctx context.Context, tenantID, userID, actorID uuid.UUID, effectiveTo time.Time) error
	Create(ctx context.Context, actorID uuid.UUID, a *entity.UserAssignment) error
	// ListByUser returns the history newest first.
	ListByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.UserAssignment, error)
}
