package employee

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// ListEmergencyContactsUseCase lists a user's emergency contacts. Used by the
// admin routes (target = path id) and by /me (target = caller).
type ListEmergencyContactsUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.EmergencyContact, error)
}
