package employee

import (
	"context"

	"github.com/google/uuid"
)

// ProfileGateUseCase answers whether a non-admin caller may use the rest of
// the API (profile complete). Used by the gating middleware.
type ProfileGateUseCase interface {
	// Execute reports whether the user's profile is complete.
	Execute(ctx context.Context, tenantID, userID uuid.UUID) (bool, error)
}
