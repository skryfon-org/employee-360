// Package usecaseinterface defines usecase interfaces (ports) for the application.
package usecaseinterface

import (
	"context"

	healthtypes "github.com/skryfon-org/employee-360/backend/internal/types/health"
)

// HealthUseCase defines the interface for checking system health.
type HealthUseCase interface {
	Execute(ctx context.Context) healthtypes.HealthResult
}
