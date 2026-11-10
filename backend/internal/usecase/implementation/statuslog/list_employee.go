package statuslog

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	statusloguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/statuslog"
)

type listEmployeeStatusLogsUseCase struct {
	repo  repository.StatusLogRepository
	users repository.UserRepository
	clock Clock
}

// NewListEmployeeStatusLogsUseCase constructs ListEmployeeStatusLogsUseCase.
func NewListEmployeeStatusLogsUseCase(repo repository.StatusLogRepository, users repository.UserRepository, clock Clock) statusloguc.ListEmployeeStatusLogsUseCase {
	if clock == nil {
		clock = RealClock
	}
	return &listEmployeeStatusLogsUseCase{repo: repo, users: users, clock: clock}
}

func (uc *listEmployeeStatusLogsUseCase) Execute(ctx context.Context, tenantID, employeeID uuid.UUID, fromStr, toStr, projectStr string) ([]*entity.DailyStatusLogGroup, error) {
	if tenantID == uuid.Nil || employeeID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	if _, err := uc.users.GetByID(ctx, tenantID, employeeID); err != nil {
		return nil, domainerrors.ErrUserNotFound
	}

	fromDate, toDate, err := ResolveDateRange(uc.clock(), fromStr, toStr)
	if err != nil {
		return nil, err
	}

	projID, err := ParseProjectFilter(projectStr)
	if err != nil {
		return nil, err
	}

	entries, err := uc.repo.ListByUserAndDateRange(ctx, tenantID, employeeID, fromDate, toDate, projID)
	if err != nil {
		return nil, err
	}

	return GroupByDate(entries), nil
}
