package entity

import (
	"time"

	"github.com/google/uuid"
)

// Activity categories for status logs not tied to a project.
const (
	ActivityCategoryMeeting  = "meeting"
	ActivityCategoryTraining = "training"
	ActivityCategoryInternal = "internal"
	ActivityCategorySupport  = "support"
	ActivityCategoryOther    = "other"
)

// AllowedActivityCategories lists valid activity categories.
var AllowedActivityCategories = []string{
	ActivityCategoryMeeting,
	ActivityCategoryTraining,
	ActivityCategoryInternal,
	ActivityCategorySupport,
	ActivityCategoryOther,
}

// IsValidActivityCategory checks if a string is a valid activity category.
func IsValidActivityCategory(cat string) bool {
	for _, c := range AllowedActivityCategories {
		if c == cat {
			return true
		}
	}
	return false
}

// StatusLogEntry represents a single status log entry.
type StatusLogEntry struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	UserID           uuid.UUID  `json:"user_id"`
	LogDate          time.Time  `json:"log_date"`
	ProjectID        *uuid.UUID `json:"project_id,omitempty"`
	ActivityCategory *string    `json:"activity_category,omitempty"`
	Hours            float64    `json:"hours"`
	Notes            string     `json:"notes"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

// StatusLogEntryWithProject joins an entry with its project's name and code.
type StatusLogEntryWithProject struct {
	StatusLogEntry
	ProjectName *string `json:"project_name,omitempty"`
	ProjectCode *string `json:"project_code,omitempty"`
}

// ProjectHoursBreakdown holds project-level hours summary.
type ProjectHoursBreakdown struct {
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectName string    `json:"project_name"`
	ProjectCode string    `json:"project_code"`
	Hours       float64   `json:"hours"`
}

// CategoryHoursBreakdown holds category-level hours summary.
type CategoryHoursBreakdown struct {
	Category string  `json:"category"`
	Hours    float64 `json:"hours"`
}

// DailyEmployeeSummary summarizes status logging for an active employee on a given date.
type DailyEmployeeSummary struct {
	UserID          uuid.UUID                `json:"user_id"`
	FirstName       string                   `json:"first_name"`
	LastName        string                   `json:"last_name"`
	Email           string                   `json:"email"`
	DepartmentID    *uuid.UUID               `json:"department_id,omitempty"`
	DepartmentName  *string                  `json:"department_name,omitempty"`
	Logged          bool                     `json:"logged"`
	TotalHours      float64                  `json:"total_hours"`
	HoursByProject  []ProjectHoursBreakdown  `json:"hours_by_project"`
	HoursByCategory []CategoryHoursBreakdown `json:"hours_by_category"`
	// DayStatus explains the day; only DayStatusMissing means the employee failed to log.
	// Filled by the daily usecase, not the repository.
	DayStatus string `json:"day_status"`
	// HalfDay is true when approved half-day leave covers the day and nothing was logged (DayStatus is missing).
	HalfDay bool `json:"half_day"`
}

// Day status values for DailyEmployeeSummary.DayStatus.
const (
	DayStatusLogged    = "logged"
	DayStatusOnLeave   = "on_leave"
	DayStatusHoliday   = "holiday"
	DayStatusWeeklyOff = "weekly_off"
	DayStatusMissing   = "missing"
)

// MonthlyEmployeeSummary summarizes an active employee's status logging over a month.
type MonthlyEmployeeSummary struct {
	UserID         uuid.UUID
	FirstName      string
	LastName       string
	Email          string
	DepartmentID   *uuid.UUID
	DepartmentName *string
	DaysLogged     int
	// OffDaysWorked counts distinct logged dates that fall on a weekly off day.
	OffDaysWorked int
	// LoggedDates are the distinct dates (UTC midnight) with at least one entry in
	// the month, including weekly off days, holidays and leave days.
	LoggedDates []time.Time
	// LeaveDays is the working days of the month spent on approved leave (half day 0.5);
	// filled by the monthly usecase, not the repository.
	LeaveDays float64
	// ExpectedDays is working days (net of public holidays) minus LeaveDays, floor 0;
	// filled by the monthly usecase.
	ExpectedDays float64
	// MissingDays is the expected working days with no log: weekly off days and public
	// holidays never count, a logged day (even on an off day or while on leave) is never
	// missing, and approved leave on an unlogged day reduces it by the leave fraction.
	// Filled by the monthly usecase.
	MissingDays     float64
	TotalHours      float64
	HoursByProject  []ProjectHoursBreakdown
	HoursByCategory []CategoryHoursBreakdown
}

// MonthlyStatusLogReport is the org-wide monthly report.
type MonthlyStatusLogReport struct {
	Month          string // YYYY-MM
	WorkingDays    int
	TotalEmployees int
	// LoggedEmployees counts employees with at least one logged day.
	LoggedEmployees int
	// NotLoggedEmployees counts employees with MissingDays > 0 (not merely employees with no logs).
	NotLoggedEmployees int
	TotalHours         float64
	Employees          []*MonthlyEmployeeSummary
}

// DailyStatusLogGroup groups entries for a single date with a total.
type DailyStatusLogGroup struct {
	Date       time.Time                    `json:"date"`
	TotalHours float64                      `json:"total_hours"`
	Entries    []*StatusLogEntryWithProject `json:"entries"`
}

// MyDayStatus is the caller's status for one calendar day (see DayStatus* constants).
type MyDayStatus struct {
	Date time.Time
	// DayStatus is one of logged, on_leave, holiday, weekly_off, missing.
	DayStatus string
	// HalfDay is true when every covering approved leave is a half day and nothing was logged (DayStatus is then missing).
	HalfDay bool
	// HolidayName is set when DayStatus is holiday.
	HolidayName string
	// LeaveTypeName is set when DayStatus is on_leave, or missing with HalfDay.
	LeaveTypeName string
	TotalHours    float64
}
