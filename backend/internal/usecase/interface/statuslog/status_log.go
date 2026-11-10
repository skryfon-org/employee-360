package statuslog

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/types/statuslog"
)

// CreateStatusLogUseCase creates a new status log entry for the caller.
type CreateStatusLogUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, req statuslog.CreateStatusLogRequest) (*entity.StatusLogEntryWithProject, error)
}

// UpdateStatusLogUseCase updates an existing status log entry owned by the caller.
type UpdateStatusLogUseCase interface {
	Execute(ctx context.Context, tenantID, actorID, id uuid.UUID, req statuslog.UpdateStatusLogRequest) (*entity.StatusLogEntryWithProject, error)
}

// DeleteStatusLogUseCase soft-deletes a status log entry owned by the caller.
type DeleteStatusLogUseCase interface {
	Execute(ctx context.Context, tenantID, actorID, id uuid.UUID) error
}

// ListMyStatusLogsUseCase lists the caller's entries grouped by date with per-day totals.
type ListMyStatusLogsUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID, fromStr, toStr, projectStr string) ([]*entity.DailyStatusLogGroup, error)
}

// GetDailyStatusLogsUseCase returns the admin daily summary for all active employees.
type GetDailyStatusLogsUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, dateStr, deptStr, projStr string) ([]*entity.DailyEmployeeSummary, error)
}

// GetMonthlyStatusLogsUseCase returns the admin org-wide monthly report for all active employees.
type GetMonthlyStatusLogsUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, monthStr, deptStr, projStr string) (*entity.MonthlyStatusLogReport, error)
}

// ListEmployeeStatusLogsUseCase lists a specific employee's entries grouped by date.
type ListEmployeeStatusLogsUseCase interface {
	Execute(ctx context.Context, tenantID, employeeID uuid.UUID, fromStr, toStr, projectStr string) ([]*entity.DailyStatusLogGroup, error)
}

// ListMyDayStatusesUseCase returns the caller's per-day status (logged, on_leave,
// holiday, weekly_off, missing) for every calendar day in the range up to today.
type ListMyDayStatusesUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID, fromStr, toStr string) ([]*entity.MyDayStatus, error)
}
