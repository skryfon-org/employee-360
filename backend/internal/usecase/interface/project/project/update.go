package project

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
)

// UpdateProjectUseCase patches a project, including status changes.
type UpdateProjectUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, in prjtypes.UpdateProjectInput) (*entity.Project, error)
}
