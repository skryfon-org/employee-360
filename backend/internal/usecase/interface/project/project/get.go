package project

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// GetProjectUseCase returns one project.
type GetProjectUseCase interface {
	Execute(ctx context.Context, tenantID, id uuid.UUID) (*entity.Project, error)
}
