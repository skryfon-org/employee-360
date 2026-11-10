package statuslog

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	statusloguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/statuslog"
)

type listMyDayStatusesUseCase struct {
	repo  repository.StatusLogRepository
	off   repository.WeeklyOffDaysReader
	leave repository.LeaveRequestRepository
	hols  repository.HolidayRepository
	clock Clock
}

// NewListMyDayStatusesUseCase constructs ListMyDayStatusesUseCase.
func NewListMyDayStatusesUseCase(repo repository.StatusLogRepository, off repository.WeeklyOffDaysReader, leave repository.LeaveRequestRepository, hols repository.HolidayRepository, clock Clock) statusloguc.ListMyDayStatusesUseCase {
	if clock == nil {
		clock = RealClock
	}
	return &listMyDayStatusesUseCase{repo: repo, off: off, leave: leave, hols: hols, clock: clock}
}

// Execute returns one item per calendar day in [from, to], oldest first. Days after
// today are omitted (the range is clamped to today), so a range entirely in the
// future yields an empty list. Classification uses ClassifyDay, the same precedence
// as the admin daily summary.
func (uc *listMyDayStatusesUseCase) Execute(ctx context.Context, tenantID, userID uuid.UUID, fromStr, toStr string) ([]*entity.MyDayStatus, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	now := uc.clock()
	from, to, err := ResolveDateRange(now, fromStr, toStr)
	if err != nil {
		return nil, err
	}
	if today := DateOnly(now); to.After(today) {
		to = today
	}
	if from.After(to) {
		return []*entity.MyDayStatus{}, nil
	}

	offDays, err := ucshared.ResolveOffDays(ctx, uc.off, tenantID)
	if err != nil {
		return nil, err
	}

	hoursOn := map[time.Time]float64{}
	entries, err := uc.repo.ListByUserAndDateRange(ctx, tenantID, userID, from, to, nil)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		k := DateOnly(e.LogDate)
		hoursOn[k] = RoundHours(hoursOn[k] + e.Hours)
	}

	hols, err := ucshared.PublicHolidays(ctx, uc.hols, tenantID, from, to)
	if err != nil {
		return nil, err
	}
	holidayOn := make(map[time.Time]string, len(hols))
	for _, h := range hols {
		k := DateOnly(h.Date)
		if _, ok := holidayOn[k]; !ok {
			holidayOn[k] = h.Name
		}
	}

	// The repository scopes these to the tenant and user, approved only, overlapping [from, to].
	reqs, _, err := uc.leave.ListByUser(ctx, tenantID, userID, repository.LeaveRequestFilter{Status: entity.LeaveStatusApproved, From: &from, To: &to, OldestFirst: true}, ucshared.NoLimit, 0)
	if err != nil {
		return nil, err
	}

	out := make([]*entity.MyDayStatus, 0, inclusiveDayCount(from, to))
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		hours, logged := hoursOn[d]
		holName, isHoliday := holidayOn[d]
		isOff := IsWeeklyOffDay(offDays, d)

		var l dayLeave
		onLeave := false
		if !isHoliday && !isOff {
			l, onLeave = leaveCoverage(d, offDays, reqs)[userID]
		}

		item := &entity.MyDayStatus{Date: d, TotalHours: hours}
		item.DayStatus, item.HalfDay = ClassifyDay(logged, isHoliday, isOff, onLeave, l.half)
		switch {
		case item.DayStatus == entity.DayStatusHoliday:
			item.HolidayName = holName
		case item.DayStatus == entity.DayStatusOnLeave || item.HalfDay:
			// on_leave, or missing with half_day (half-day leave, nothing logged).
			item.LeaveTypeName = l.typeName
		}
		out = append(out, item)
	}
	return out, nil
}
