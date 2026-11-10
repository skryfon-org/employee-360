package employee

import (
	"context"

	"github.com/google/uuid"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// DeleteEmergencyContactUseCase removes a contact of the target user.
type DeleteEmergencyContactUseCase interface {
	Execute(ctx context.Context, actor emptypes.Actor, userID, contactID uuid.UUID) error
}
