package employee

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// UpdateEmergencyContactUseCase replaces a contact of the target user.
type UpdateEmergencyContactUseCase interface {
	Execute(ctx context.Context, actor emptypes.Actor, userID, contactID uuid.UUID, in emptypes.EmergencyContactInput) (*entity.EmergencyContact, error)
}
