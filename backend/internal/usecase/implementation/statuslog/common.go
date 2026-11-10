package statuslog

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/types/statuslog"
)

// Clock returns the current reference time in UTC.
type Clock func() time.Time

// RealClock returns the current UTC time.
func RealClock() time.Time {
	return time.Now().UTC()
}

// DateOnly truncates a time to UTC midnight.
func DateOnly(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ParseDateOnly parses a YYYY-MM-DD string into a UTC midnight time.Time.
func ParseDateOnly(s string) (time.Time, error) {
	t, err := time.ParseInLocation(statuslog.DateLayout, strings.TrimSpace(s), time.UTC)
	if err != nil {
		return time.Time{}, domainerrors.ErrInvalidStatusLogDate
	}
	return DateOnly(t), nil
}

// ValidateLogDateWindow verifies that logDate is within [today-6, today].
// Today and today-6 are accepted; today-7 and future dates are rejected.
func ValidateLogDateWindow(logDate, now time.Time) error {
	today := DateOnly(now)
	earliest := today.AddDate(0, 0, -6)
	if logDate.After(today) {
		return domainerrors.ErrFutureDate
	}
	if logDate.Before(earliest) {
		return domainerrors.ErrLogWindowClosed
	}
	return nil
}

// RoundHours rounds float64 hours to 2 decimal places.
func RoundHours(h float64) float64 {
	return math.Round(h*100) / 100
}

// GroupByDate groups status log entries by calendar date, calculating per-day totals.
// Returns groups ordered from newest date to oldest.
func GroupByDate(entries []*entity.StatusLogEntryWithProject) []*entity.DailyStatusLogGroup {
	type dayKey struct {
		year  int
		month time.Month
		day   int
	}

	groupsMap := make(map[dayKey]*entity.DailyStatusLogGroup)
	var keys []dayKey

	for _, e := range entries {
		d := DateOnly(e.LogDate)
		k := dayKey{year: d.Year(), month: d.Month(), day: d.Day()}
		group, exists := groupsMap[k]
		if !exists {
			group = &entity.DailyStatusLogGroup{
				Date:       d,
				TotalHours: 0,
				Entries:    []*entity.StatusLogEntryWithProject{},
			}
			groupsMap[k] = group
			keys = append(keys, k)
		}
		group.Entries = append(group.Entries, e)
		group.TotalHours = RoundHours(group.TotalHours + e.Hours)
	}

	sort.Slice(keys, func(i, j int) bool {
		tI := time.Date(keys[i].year, keys[i].month, keys[i].day, 0, 0, 0, 0, time.UTC)
		tJ := time.Date(keys[j].year, keys[j].month, keys[j].day, 0, 0, 0, 0, time.UTC)
		return tI.After(tJ) // newest first
	})

	result := make([]*entity.DailyStatusLogGroup, 0, len(keys))
	for _, k := range keys {
		result = append(result, groupsMap[k])
	}
	return result
}

// MaxStatusLogRangeDays is the maximum span in days allowed for date range queries (inclusive).
const MaxStatusLogRangeDays = 366

// ResolveDateRange parses and validates from/to date query parameters.
// If omitted, defaults to [today-6, today].
// Returns an error if dates are invalid, from > to, or the range exceeds MaxStatusLogRangeDays.
func ResolveDateRange(now time.Time, fromStr, toStr string) (time.Time, time.Time, error) {
	today := DateOnly(now)
	fromDate := today.AddDate(0, 0, -6)
	toDate := today

	if strings.TrimSpace(fromStr) != "" {
		f, err := ParseDateOnly(fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		fromDate = f
	}

	if strings.TrimSpace(toStr) != "" {
		t, err := ParseDateOnly(toStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		toDate = t
	}

	if fromDate.After(toDate) {
		return time.Time{}, time.Time{}, domainerrors.ErrInvalidStatusLogDate
	}

	days := int(toDate.Sub(fromDate)/(24*time.Hour)) + 1
	if days > MaxStatusLogRangeDays {
		return time.Time{}, time.Time{}, domainerrors.ErrStatusLogDateRangeExceeded
	}

	return fromDate, toDate, nil
}

// ParseProjectFilter parses an optional project query string parameter.
func ParseProjectFilter(projectStr string) (*uuid.UUID, error) {
	if strings.TrimSpace(projectStr) == "" {
		return nil, nil
	}
	p, err := uuid.Parse(strings.TrimSpace(projectStr))
	if err != nil {
		return nil, domainerrors.ErrInvalidProjectData
	}
	return &p, nil
}

// ResolveActivity resolves and validates the project ID and activity category.
// Exactly one of a valid project ID (non-nil UUID) or a valid activity category must be set.
func ResolveActivity(projectID *uuid.UUID, activityCategory *string) (*uuid.UUID, *string, error) {
	var hasProject bool
	var effectiveProjectID *uuid.UUID
	if projectID != nil && *projectID != uuid.Nil {
		hasProject = true
		effectiveProjectID = projectID
	}

	var category *string
	if activityCategory != nil && strings.TrimSpace(*activityCategory) != "" {
		c := strings.TrimSpace(*activityCategory)
		category = &c
	}
	hasCategory := category != nil

	if (hasProject && hasCategory) || (!hasProject && !hasCategory) {
		return nil, nil, domainerrors.ErrProjectOrCategoryRequired
	}

	if hasCategory && !entity.IsValidActivityCategory(*category) {
		return nil, nil, domainerrors.ErrInvalidActivityCategory
	}

	return effectiveProjectID, category, nil
}
