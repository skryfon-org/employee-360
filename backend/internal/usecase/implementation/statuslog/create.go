package statuslog

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/types/statuslog"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	statusloguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/statuslog"
)

type createStatusLogUseCase struct {
	repo        repository.StatusLogRepository
	allocations repository.ProjectAllocationRepository
	transactor  ucshared.Transactor
	clock       Clock
}

// NewCreateStatusLogUseCase constructs CreateStatusLogUseCase.
func NewCreateStatusLogUseCase(
	repo repository.StatusLogRepository,
	allocations repository.ProjectAllocationRepository,
	transactor ucshared.Transactor,
	clock Clock,
) statusloguc.CreateStatusLogUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	if clock == nil {
		clock = RealClock
	}
	return &createStatusLogUseCase{
		repo:        repo,
		allocations: allocations,
		transactor:  transactor,
		clock:       clock,
	}
}

func (uc *createStatusLogUseCase) Execute(ctx context.Context, tenantID, actorID uuid.UUID, req statuslog.CreateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	logDate, err := ParseDateOnly(req.LogDate)
	if err != nil {
		return nil, err
	}

	if err := ValidateLogDateWindow(logDate, uc.clock()); err != nil {
		return nil, err
	}

	hours := RoundHours(req.Hours)
	if hours <= 0 || hours > 24 {
		return nil, domainerrors.ErrInvalidStatusLogHours
	}

	projectID, category, err := ResolveActivity(req.ProjectID, req.ActivityCategory)
	if err != nil {
		return nil, err
	}

	if projectID != nil {
		allocs, err := uc.allocations.ListActiveByUser(ctx, tenantID, actorID, logDate)
		if err != nil {
			return nil, err
		}
		var allocated bool
		for _, a := range allocs {
			if a.ProjectID == *projectID {
				allocated = true
				break
			}
		}
		if !allocated {
			return nil, domainerrors.ErrNoActiveAllocation
		}
	}

	entryID := uuid.New()
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.repo.LockUserDay(txCtx, tenantID, actorID, logDate); err != nil {
			return err
		}

		dayTotal, err := uc.repo.GetDayTotalHours(txCtx, tenantID, actorID, logDate, uuid.Nil)
		if err != nil {
			return err
		}
		if RoundHours(dayTotal+hours) > 24.0 {
			return domainerrors.ErrDailyHoursExceeded
		}

		entry := &entity.StatusLogEntry{
			ID:               entryID,
			TenantID:         tenantID,
			UserID:           actorID,
			LogDate:          logDate,
			ProjectID:        projectID,
			ActivityCategory: category,
			Hours:            hours,
			Notes:            strings.TrimSpace(req.Notes),
		}

		return uc.repo.Create(txCtx, tenantID, actorID, entry)
	})
	if err != nil {
		return nil, err
	}

	return uc.repo.GetByIDWithProject(ctx, tenantID, entryID)
}
