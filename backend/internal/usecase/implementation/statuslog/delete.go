package statuslog

import (
	"context"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	statusloguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/statuslog"
)

type deleteStatusLogUseCase struct {
	repo       repository.StatusLogRepository
	transactor ucshared.Transactor
	clock      Clock
}

// NewDeleteStatusLogUseCase constructs DeleteStatusLogUseCase.
func NewDeleteStatusLogUseCase(
	repo repository.StatusLogRepository,
	transactor ucshared.Transactor,
	clock Clock,
) statusloguc.DeleteStatusLogUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	if clock == nil {
		clock = RealClock
	}
	return &deleteStatusLogUseCase{
		repo:       repo,
		transactor: transactor,
		clock:      clock,
	}
}

func (uc *deleteStatusLogUseCase) Execute(ctx context.Context, tenantID, actorID, id uuid.UUID) error {
	if tenantID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}

	existing, err := uc.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if existing.UserID != actorID {
		return domainerrors.ErrStatusLogNotFound
	}

	if err := ValidateLogDateWindow(existing.LogDate, uc.clock()); err != nil {
		return err
	}

	return uc.repo.SoftDelete(ctx, tenantID, id, actorID)
}
