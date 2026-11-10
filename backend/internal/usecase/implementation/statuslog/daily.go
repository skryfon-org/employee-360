package statuslog

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	statusloguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/statuslog"
)

type getDailyStatusLogsUseCase struct {
	repo  repository.StatusLogRepository
	off   repository.WeeklyOffDaysReader
	leave repository.LeaveRequestRepository
	hols  repository.HolidayRepository
	clock Clock
}

// NewGetDailyStatusLogsUseCase constructs GetDailyStatusLogsUseCase.
func NewGetDailyStatusLogsUseCase(repo repository.StatusLogRepository, off repository.WeeklyOffDaysReader, leave repository.LeaveRequestRepository, hols repository.HolidayRepository, clock Clock) statusloguc.GetDailyStatusLogsUseCase {
	if clock == nil {
		clock = RealClock
	}
	return &getDailyStatusLogsUseCase{repo: repo, off: off, leave: leave, hols: hols, clock: clock}
}

func (uc *getDailyStatusLogsUseCase) Execute(ctx context.Context, tenantID uuid.UUID, dateStr, deptStr, projStr string) ([]*entity.DailyEmployeeSummary, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	targetDate := DateOnly(uc.clock())
	if strings.TrimSpace(dateStr) != "" {
		d, err := ParseDateOnly(dateStr)
		if err != nil {
			return nil, err
		}
		targetDate = d
	}

	var deptID *uuid.UUID
	if strings.TrimSpace(deptStr) != "" {
		d, err := uuid.Parse(strings.TrimSpace(deptStr))
		if err != nil {
			return nil, domainerrors.ErrDepartmentNotFound
		}
		deptID = &d
	}

	var projID *uuid.UUID
	if strings.TrimSpace(projStr) != "" {
		p, err := uuid.Parse(strings.TrimSpace(projStr))
		if err != nil {
			return nil, domainerrors.ErrInvalidProjectData
		}
		projID = &p
	}

	rows, err := uc.repo.DailySummary(ctx, tenantID, targetDate, deptID, projID)
	if err != nil {
		return nil, err
	}
	if err := uc.fillDayStatus(ctx, tenantID, targetDate, rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// fillDayStatus sets DayStatus/HalfDay on each row. Precedence: see ClassifyDay (approved
// half-day leave with no log is missing with HalfDay set). Leave on a holiday or weekly
// off day is not on_leave, matching entity.CountLeaveDays.
func (uc *getDailyStatusLogsUseCase) fillDayStatus(ctx context.Context, tenantID uuid.UUID, date time.Time, rows []*entity.DailyEmployeeSummary) error {
	if len(rows) == 0 {
		return nil
	}
	offDays, err := ucshared.ResolveOffDays(ctx, uc.off, tenantID)
	if err != nil {
		return err
	}
	isOff := IsWeeklyOffDay(offDays, date)
	hols, err := ucshared.PublicHolidays(ctx, uc.hols, tenantID, date, date)
	if err != nil {
		return err
	}
	isHoliday := len(hols) > 0
	var covered map[uuid.UUID]dayLeave
	if !isOff && !isHoliday {
		// The repository returns this tenant's approved requests overlapping date.
		reqs, _, err := uc.leave.List(ctx, tenantID, repository.LeaveRequestFilter{Status: entity.LeaveStatusApproved, From: &date, To: &date}, ucshared.NoLimit, 0)
		if err != nil {
			return err
		}
		covered = leaveCoverage(date, offDays, reqs)
	}
	for _, row := range rows {
		l, onLeave := covered[row.UserID]
		row.DayStatus, row.HalfDay = ClassifyDay(row.Logged, isHoliday, isOff, onLeave, l.half)
	}
	return nil
}
