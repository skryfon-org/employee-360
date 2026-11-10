package project

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	prjuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/project"
)

type getProjectUseCase struct{ repo repository.ProjectRepository }

// NewGetProjectUseCase creates a GetProjectUseCase.
func NewGetProjectUseCase(repo repository.ProjectRepository) prjuc.GetProjectUseCase {
	return &getProjectUseCase{repo: repo}
}

func (uc *getProjectUseCase) Execute(c context.Context, tenantID, id uuid.UUID) (*entity.Project, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	return uc.repo.GetByID(c, tenantID, id)
}
