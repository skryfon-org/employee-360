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

type updateStatusLogUseCase struct {
	repo        repository.StatusLogRepository
	allocations repository.ProjectAllocationRepository
	transactor  ucshared.Transactor
	clock       Clock
}

// NewUpdateStatusLogUseCase constructs UpdateStatusLogUseCase.
func NewUpdateStatusLogUseCase(
	repo repository.StatusLogRepository,
	allocations repository.ProjectAllocationRepository,
	transactor ucshared.Transactor,
	clock Clock,
) statusloguc.UpdateStatusLogUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	if clock == nil {
		clock = RealClock
	}
	return &updateStatusLogUseCase{
		repo:        repo,
		allocations: allocations,
		transactor:  transactor,
		clock:       clock,
	}
}

func (uc *updateStatusLogUseCase) Execute(ctx context.Context, tenantID, actorID, id uuid.UUID, req statuslog.UpdateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	existing, err := uc.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if existing.UserID != actorID {
		return nil, domainerrors.ErrStatusLogNotFound
	}

	now := uc.clock()
	if err := ValidateLogDateWindow(existing.LogDate, now); err != nil {
		return nil, err
	}

	effectiveDate := existing.LogDate
	if req.LogDate != nil && strings.TrimSpace(*req.LogDate) != "" {
		newDate, err := ParseDateOnly(*req.LogDate)
		if err != nil {
			return nil, err
		}
		if err := ValidateLogDateWindow(newDate, now); err != nil {
			return nil, err
		}
		effectiveDate = newDate
	}

	effectiveHours := existing.Hours
	if req.Hours != nil {
		h := RoundHours(*req.Hours)
		if h <= 0 || h > 24 {
			return nil, domainerrors.ErrInvalidStatusLogHours
		}
		effectiveHours = h
	}

	effectiveProjectID := existing.ProjectID
	effectiveCategory := existing.ActivityCategory

	if req.ProjectID != nil || req.ActivityCategory != nil {
		projID, cat, err := ResolveActivity(req.ProjectID, req.ActivityCategory)
		if err != nil {
			return nil, err
		}
		effectiveProjectID = projID
		effectiveCategory = cat
	}

	if effectiveProjectID != nil {
		allocs, err := uc.allocations.ListActiveByUser(ctx, tenantID, actorID, effectiveDate)
		if err != nil {
			return nil, err
		}
		var allocated bool
		for _, a := range allocs {
			if a.ProjectID == *effectiveProjectID {
				allocated = true
				break
			}
		}
		if !allocated {
			return nil, domainerrors.ErrNoActiveAllocation
		}
	}

	effectiveNotes := existing.Notes
	if req.Notes != nil {
		effectiveNotes = strings.TrimSpace(*req.Notes)
	}

	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.repo.LockUserDay(txCtx, tenantID, actorID, effectiveDate); err != nil {
			return err
		}

		dayTotal, err := uc.repo.GetDayTotalHours(txCtx, tenantID, actorID, effectiveDate, id)
		if err != nil {
			return err
		}
		if RoundHours(dayTotal+effectiveHours) > 24.0 {
			return domainerrors.ErrDailyHoursExceeded
		}

		existing.LogDate = effectiveDate
		existing.ProjectID = effectiveProjectID
		existing.ActivityCategory = effectiveCategory
		existing.Hours = effectiveHours
		existing.Notes = effectiveNotes

		return uc.repo.Update(txCtx, tenantID, actorID, existing)
	})
	if err != nil {
		return nil, err
	}

	return uc.repo.GetByIDWithProject(ctx, tenantID, id)
}
