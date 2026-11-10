package employee

import (
	"context"

	"github.com/google/uuid"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// GetMeUseCase returns the caller's identity, organization and onboarding state.
type GetMeUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID) (*emptypes.Me, error)
}
