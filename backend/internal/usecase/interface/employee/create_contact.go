package employee

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// CreateEmergencyContactUseCase adds a contact for the target user.
type CreateEmergencyContactUseCase interface {
	Execute(ctx context.Context, actor emptypes.Actor, userID uuid.UUID, in emptypes.EmergencyContactInput) (*entity.EmergencyContact, error)
}
