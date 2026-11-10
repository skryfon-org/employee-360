package project

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
)

// CreateProjectUseCase creates a project in the caller's tenant.
type CreateProjectUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, in prjtypes.CreateProjectInput) (*entity.Project, error)
}
