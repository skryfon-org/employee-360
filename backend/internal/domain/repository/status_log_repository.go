package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// StatusLogRepository manages persistence for status log entries and daily summaries.
// Every method enforces tenant isolation via the explicit tenantID parameter.
type StatusLogRepository interface {
	LockUserDay(ctx context.Context, tenantID, userID uuid.UUID, logDate time.Time) error
	Create(ctx context.Context, tenantID, actorID uuid.UUID, entry *entity.StatusLogEntry) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.StatusLogEntry, error)
	GetByIDWithProject(ctx context.Context, tenantID, id uuid.UUID) (*entity.StatusLogEntryWithProject, error)
	Update(ctx context.Context, tenantID, actorID uuid.UUID, entry *entity.StatusLogEntry) error
	SoftDelete(ctx context.Context, tenantID, id, actorID uuid.UUID) error
	GetDayTotalHours(ctx context.Context, tenantID, userID uuid.UUID, logDate time.Time, excludeID uuid.UUID) (float64, error)
	ListByUserAndDateRange(ctx context.Context, tenantID, userID uuid.UUID, from, to time.Time, projectID *uuid.UUID) ([]*entity.StatusLogEntryWithProject, error)
	DailySummary(ctx context.Context, tenantID uuid.UUID, date time.Time, departmentID, projectID *uuid.UUID) ([]*entity.DailyEmployeeSummary, error)
	// MonthlySummary aggregates every active employee over the inclusive [from, to] range.
	// offDays (0 = Sunday .. 6 = Saturday) drives OffDaysWorked; hours on off days still count.
	MonthlySummary(ctx context.Context, tenantID uuid.UUID, from, to time.Time, offDays []int, departmentID, projectID *uuid.UUID) ([]*entity.MonthlyEmployeeSummary, error)
}
