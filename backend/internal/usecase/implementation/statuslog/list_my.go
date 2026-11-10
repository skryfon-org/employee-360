package statuslog

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	statusloguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/statuslog"
)

type listMyStatusLogsUseCase struct {
	repo  repository.StatusLogRepository
	clock Clock
}

// NewListMyStatusLogsUseCase constructs ListMyStatusLogsUseCase.
func NewListMyStatusLogsUseCase(repo repository.StatusLogRepository, clock Clock) statusloguc.ListMyStatusLogsUseCase {
	if clock == nil {
		clock = RealClock
	}
	return &listMyStatusLogsUseCase{repo: repo, clock: clock}
}

func (uc *listMyStatusLogsUseCase) Execute(ctx context.Context, tenantID, userID uuid.UUID, fromStr, toStr, projectStr string) ([]*entity.DailyStatusLogGroup, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	fromDate, toDate, err := ResolveDateRange(uc.clock(), fromStr, toStr)
	if err != nil {
		return nil, err
	}

	projID, err := ParseProjectFilter(projectStr)
	if err != nil {
		return nil, err
	}

	entries, err := uc.repo.ListByUserAndDateRange(ctx, tenantID, userID, fromDate, toDate, projID)
	if err != nil {
		return nil, err
	}

	return GroupByDate(entries), nil
}
