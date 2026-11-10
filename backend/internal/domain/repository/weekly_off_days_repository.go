package repository

import (
	"context"

	"github.com/google/uuid"
)

// WeeklyOffDaysReader exposes a tenant's weekly off days (0 = Sunday .. 6 = Saturday).
// The tenantID is explicit; a tenant without a settings row yields the default {0,6}.
type WeeklyOffDaysReader interface {
	WeeklyOffDays(ctx context.Context, tenantID uuid.UUID) ([]int, error)
}
