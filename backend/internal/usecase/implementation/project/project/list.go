package project

import (
	"context"

	"github.com/google/uuid"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/types/pagination"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	prjuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/project"
)

type listProjectsUseCase struct{ repo repository.ProjectRepository }

// NewListProjectsUseCase creates a ListProjectsUseCase.
func NewListProjectsUseCase(repo repository.ProjectRepository) prjuc.ListProjectsUseCase {
	return &listProjectsUseCase{repo: repo}
}

func (uc *listProjectsUseCase) Execute(c context.Context, tenantID uuid.UUID, q prjtypes.ListProjectsQuery) (*prjtypes.ListProjectsResult, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if q.Status != "" && !validStatus(q.Status) {
		return nil, ErrStatusInvalid
	}
	page, size, offset := pagination.Normalize(q.Page, q.PageSize)
	items, total, err := uc.repo.List(c, tenantID, repository.ProjectFilter{Search: q.Search, Status: q.Status}, size, offset)
	if err != nil {
		return nil, err
	}
	return &prjtypes.ListProjectsResult{Projects: items, Total: total, Page: page, PageSize: size}, nil
}
