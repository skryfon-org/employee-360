package ucshared

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
)

// NoLimit is the limit value repositories read as "no pagination limit".
const NoLimit = 0

// ResolveOffDays returns the tenant's weekly off days (0 = Sunday .. 6 = Saturday),
// falling back to the platform default when none are configured.
func ResolveOffDays(ctx context.Context, reader repository.WeeklyOffDaysReader, tenantID uuid.UUID) ([]int, error) {
	d, err := reader.WeeklyOffDays(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(d) == 0 {
		return entity.DefaultWeeklyOffDays(), nil
	}
	return d, nil
}

// PublicHolidays returns the tenant's public-category holidays with from <= date <= to
// (both inclusive). Only public holidays (category is_public) are company days off;
// optional and other categories are working days.
func PublicHolidays(ctx context.Context, repo repository.HolidayRepository, tenantID uuid.UUID, from, to time.Time) ([]*entity.Holiday, error) {
	hs, err := repo.ListByDateRange(ctx, tenantID, from, to.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	return FilterPublicHolidays(hs), nil
}

// FilterPublicHolidays keeps only the holidays whose category is public.
func FilterPublicHolidays(hs []*entity.Holiday) []*entity.Holiday {
	out := make([]*entity.Holiday, 0, len(hs))
	for _, h := range hs {
		if h.CategoryIsPublic {
			out = append(out, h)
		}
	}
	return out
}

// PublicHolidayDates is PublicHolidays reduced to the holiday dates.
func PublicHolidayDates(ctx context.Context, repo repository.HolidayRepository, tenantID uuid.UUID, from, to time.Time) ([]time.Time, error) {
	hs, err := PublicHolidays(ctx, repo, tenantID, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]time.Time, 0, len(hs))
	for _, h := range hs {
		out = append(out, h.Date)
	}
	return out, nil
}
