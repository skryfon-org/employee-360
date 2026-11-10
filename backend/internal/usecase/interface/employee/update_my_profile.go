package employee

import (
	"context"

	"github.com/google/uuid"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// UpdateMyProfileUseCase updates the caller's self-service profile fields and
// recomputes completion.
type UpdateMyProfileUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID, in emptypes.UpdateMyProfileInput) (*emptypes.MyProfile, error)
}
