package employee

import (
	"context"

	"github.com/google/uuid"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// GetMyProfileUseCase returns the caller's profile, contacts and completion state.
type GetMyProfileUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID) (*emptypes.MyProfile, error)
}
