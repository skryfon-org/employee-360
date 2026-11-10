package position

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
	posuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/position"
)

type getPositionUseCase struct {
	repo repository.PositionRepository
}

// NewGetPositionUseCase creates a new GetPositionUseCase.
func NewGetPositionUseCase(repo repository.PositionRepository) posuc.GetPositionUseCase {
	return &getPositionUseCase{repo: repo}
}

func (uc *getPositionUseCase) Execute(c context.Context, tenantID uuid.UUID, input postypes.GetPositionQuery) (*entity.Position, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if input.ID == uuid.Nil {
		return nil, domainerrors.ErrPositionNotFound
	}
	return uc.repo.GetByID(c, tenantID, input.ID)
}
