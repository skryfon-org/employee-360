package project

import (
	"context"

	"github.com/google/uuid"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	prjuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/project"
)

type suggestProjectCodeUseCase struct{ repo repository.ProjectRepository }

// NewSuggestProjectCodeUseCase creates a SuggestProjectCodeUseCase.
func NewSuggestProjectCodeUseCase(repo repository.ProjectRepository) prjuc.SuggestProjectCodeUseCase {
	return &suggestProjectCodeUseCase{repo: repo}
}

func (uc *suggestProjectCodeUseCase) Execute(c context.Context, tenantID uuid.UUID, name string) (string, error) {
	if tenantID == uuid.Nil {
		return "", domainerrors.ErrUnauthorized
	}
	n, err := normalizeName(name)
	if err != nil {
		return "", err
	}
	code, _, err := firstFreeCode(c, uc.repo, tenantID, n, 1)
	return code, err
}
