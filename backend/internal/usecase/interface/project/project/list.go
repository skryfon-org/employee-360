package project

import (
	"context"

	"github.com/google/uuid"

	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
)

// ListProjectsUseCase lists projects with search, status filter and paging.
type ListProjectsUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, q prjtypes.ListProjectsQuery) (*prjtypes.ListProjectsResult, error)
}
