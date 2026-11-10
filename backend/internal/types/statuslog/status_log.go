package statuslog

import (
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// DateLayout is the wire format for date-only fields.
const DateLayout = "2006-01-02"

// CreateStatusLogRequest is the body for POST /api/v1/me/status-logs.
type CreateStatusLogRequest struct {
	LogDate          string     `json:"log_date" binding:"required" example:"2026-10-27"`
	ProjectID        *uuid.UUID `json:"project_id,omitempty"`
	ActivityCategory *string    `json:"activity_category,omitempty" enums:"meeting,training,internal,support,other"`
	Hours            float64    `json:"hours" binding:"required" example:"4.50"`
	Notes            string     `json:"notes" example:"Worked on task XYZ"`
}

// UpdateStatusLogRequest is the body for PATCH /api/v1/me/status-logs/:id.
type UpdateStatusLogRequest struct {
	LogDate          *string    `json:"log_date,omitempty" example:"2026-10-27"`
	ProjectID        *uuid.UUID `json:"project_id,omitempty"`
	ActivityCategory *string    `json:"activity_category,omitempty" enums:"meeting,training,internal,support,other"`
	Hours            *float64   `json:"hours,omitempty" example:"5.00"`
	Notes            *string    `json:"notes,omitempty" example:"Updated task notes"`
}

// StatusLogEntryResponse represents a status log entry.
type StatusLogEntryResponse struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	UserID           uuid.UUID  `json:"user_id"`
	LogDate          string     `json:"log_date"`
	ProjectID        *uuid.UUID `json:"project_id,omitempty"`
	ProjectName      *string    `json:"project_name,omitempty"`
	ProjectCode      *string    `json:"project_code,omitempty"`
	ActivityCategory *string    `json:"activity_category,omitempty"`
	Hours            float64    `json:"hours"`
	Notes            string     `json:"notes"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// DailyStatusLogGroupResponse groups entries for one date with a per-day total.
type DailyStatusLogGroupResponse struct {
	Date       string                   `json:"date"`
	TotalHours float64                  `json:"total_hours"`
	Entries    []StatusLogEntryResponse `json:"entries"`
}

// ProjectHoursBreakdownResponse holds project hours breakdown.
type ProjectHoursBreakdownResponse struct {
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectName string    `json:"project_name"`
	ProjectCode string    `json:"project_code"`
	Hours       float64   `json:"hours"`
}

// CategoryHoursBreakdownResponse holds category hours breakdown.
type CategoryHoursBreakdownResponse struct {
	Category string  `json:"category"`
	Hours    float64 `json:"hours"`
}

// DailyEmployeeSummaryResponse is the admin summary item for an active employee on a date.
type DailyEmployeeSummaryResponse struct {
	UserID          uuid.UUID                        `json:"user_id"`
	FirstName       string                           `json:"first_name"`
	LastName        string                           `json:"last_name"`
	Email           string                           `json:"email"`
	DepartmentID    *uuid.UUID                       `json:"department_id,omitempty"`
	DepartmentName  *string                          `json:"department_name,omitempty"`
	Logged          bool                             `json:"logged"`
	TotalHours      float64                          `json:"total_hours"`
	HoursByProject  []ProjectHoursBreakdownResponse  `json:"hours_by_project"`
	HoursByCategory []CategoryHoursBreakdownResponse `json:"hours_by_category"`
	// DayStatus is one of logged, on_leave, holiday, weekly_off, missing; only missing means the employee failed to log.
	DayStatus string `json:"day_status" enums:"logged,on_leave,holiday,weekly_off,missing"`
	// HalfDay is true when approved half-day leave covers the day and nothing was logged (day_status is then missing).
	HalfDay bool `json:"half_day"`
}

// MonthlyEmployeeSummaryResponse is one employee row of the monthly report.
type MonthlyEmployeeSummaryResponse struct {
	UserID         uuid.UUID  `json:"user_id"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	Email          string     `json:"email"`
	DepartmentID   *uuid.UUID `json:"department_id,omitempty"`
	DepartmentName *string    `json:"department_name,omitempty"`
	DaysLogged     int        `json:"days_logged"`
	OffDaysWorked  int        `json:"off_days_worked"`
	// LeaveDays is working days in the month on approved leave (half day = 0.5).
	LeaveDays float64 `json:"leave_days"`
	// ExpectedDays is working days net of public holidays minus leave_days (floor 0).
	ExpectedDays float64 `json:"expected_days"`
	// MissingDays is the expected working days with no log (off days, public holidays and leave are not missing; work on an off day does not offset a missed working day).
	MissingDays     float64                          `json:"missing_days"`
	TotalHours      float64                          `json:"total_hours"`
	HoursByProject  []ProjectHoursBreakdownResponse  `json:"hours_by_project"`
	HoursByCategory []CategoryHoursBreakdownResponse `json:"hours_by_category"`
}

// MonthlyStatusLogReportResponse is the org-wide monthly status log report.
type MonthlyStatusLogReportResponse struct {
	Month string `json:"month" example:"2026-10"`
	// WorkingDays is the month's working days net of weekly off days and public holidays.
	WorkingDays     int `json:"working_days"`
	TotalEmployees  int `json:"total_employees"`
	LoggedEmployees int `json:"logged_employees"`
	// NotLoggedEmployees counts employees with missing_days > 0 (not merely employees with no logs).
	NotLoggedEmployees int                              `json:"not_logged_employees"`
	TotalHours         float64                          `json:"total_hours"`
	Employees          []MonthlyEmployeeSummaryResponse `json:"employees"`
}

// ToMonthlyStatusLogReportResponse maps the report entity to its HTTP response.
func ToMonthlyStatusLogReportResponse(r *entity.MonthlyStatusLogReport) MonthlyStatusLogReportResponse {
	emps := make([]MonthlyEmployeeSummaryResponse, 0, len(r.Employees))
	for _, e := range r.Employees {
		d := ToDailyEmployeeSummaryResponses([]*entity.DailyEmployeeSummary{{
			HoursByProject: e.HoursByProject, HoursByCategory: e.HoursByCategory,
		}})[0]
		emps = append(emps, MonthlyEmployeeSummaryResponse{
			UserID:          e.UserID,
			FirstName:       e.FirstName,
			LastName:        e.LastName,
			Email:           e.Email,
			DepartmentID:    e.DepartmentID,
			DepartmentName:  e.DepartmentName,
			DaysLogged:      e.DaysLogged,
			OffDaysWorked:   e.OffDaysWorked,
			LeaveDays:       e.LeaveDays,
			ExpectedDays:    e.ExpectedDays,
			MissingDays:     e.MissingDays,
			TotalHours:      e.TotalHours,
			HoursByProject:  d.HoursByProject,
			HoursByCategory: d.HoursByCategory,
		})
	}
	return MonthlyStatusLogReportResponse{
		Month:              r.Month,
		WorkingDays:        r.WorkingDays,
		TotalEmployees:     r.TotalEmployees,
		LoggedEmployees:    r.LoggedEmployees,
		NotLoggedEmployees: r.NotLoggedEmployees,
		TotalHours:         r.TotalHours,
		Employees:          emps,
	}
}

// ToStatusLogEntryResponse maps an entity to its HTTP response.
func ToStatusLogEntryResponse(e *entity.StatusLogEntryWithProject) StatusLogEntryResponse {
	return StatusLogEntryResponse{
		ID:               e.ID,
		TenantID:         e.TenantID,
		UserID:           e.UserID,
		LogDate:          e.LogDate.UTC().Format(DateLayout),
		ProjectID:        e.ProjectID,
		ProjectName:      e.ProjectName,
		ProjectCode:      e.ProjectCode,
		ActivityCategory: e.ActivityCategory,
		Hours:            e.Hours,
		Notes:            e.Notes,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

// ToDailyStatusLogGroupResponses converts entity groups to response groups.
func ToDailyStatusLogGroupResponses(groups []*entity.DailyStatusLogGroup) []DailyStatusLogGroupResponse {
	out := make([]DailyStatusLogGroupResponse, 0, len(groups))
	for _, g := range groups {
		entries := make([]StatusLogEntryResponse, 0, len(g.Entries))
		for _, e := range g.Entries {
			entries = append(entries, ToStatusLogEntryResponse(e))
		}
		out = append(out, DailyStatusLogGroupResponse{
			Date:       g.Date.UTC().Format(DateLayout),
			TotalHours: g.TotalHours,
			Entries:    entries,
		})
	}
	return out
}

// ToDailyEmployeeSummaryResponses maps entity summaries to HTTP responses.
func ToDailyEmployeeSummaryResponses(summaries []*entity.DailyEmployeeSummary) []DailyEmployeeSummaryResponse {
	out := make([]DailyEmployeeSummaryResponse, 0, len(summaries))
	for _, s := range summaries {
		prjList := make([]ProjectHoursBreakdownResponse, 0, len(s.HoursByProject))
		for _, p := range s.HoursByProject {
			prjList = append(prjList, ProjectHoursBreakdownResponse{
				ProjectID:   p.ProjectID,
				ProjectName: p.ProjectName,
				ProjectCode: p.ProjectCode,
				Hours:       p.Hours,
			})
		}
		catList := make([]CategoryHoursBreakdownResponse, 0, len(s.HoursByCategory))
		for _, c := range s.HoursByCategory {
			catList = append(catList, CategoryHoursBreakdownResponse{
				Category: c.Category,
				Hours:    c.Hours,
			})
		}
		out = append(out, DailyEmployeeSummaryResponse{
			UserID:          s.UserID,
			FirstName:       s.FirstName,
			LastName:        s.LastName,
			Email:           s.Email,
			DepartmentID:    s.DepartmentID,
			DepartmentName:  s.DepartmentName,
			Logged:          s.Logged,
			TotalHours:      s.TotalHours,
			HoursByProject:  prjList,
			HoursByCategory: catList,
			DayStatus:       s.DayStatus,
			HalfDay:         s.HalfDay,
		})
	}
	return out
}

// MyDayStatusResponse is one calendar day of the caller's status view.
type MyDayStatusResponse struct {
	Date string `json:"date" example:"2026-10-27"`
	// DayStatus is one of logged, on_leave, holiday, weekly_off, missing; only missing means the employee has not logged.
	DayStatus string `json:"day_status" enums:"logged,on_leave,holiday,weekly_off,missing"`
	// HalfDay is true when approved half-day leave covers the day and nothing was logged (day_status is then missing).
	HalfDay bool `json:"half_day"`
	// HolidayName is set only when day_status is holiday.
	HolidayName *string `json:"holiday_name,omitempty"`
	// LeaveTypeName is set when day_status is on_leave, or missing with half_day.
	LeaveTypeName *string `json:"leave_type_name,omitempty"`
	TotalHours    float64 `json:"total_hours"`
}

// ToMyDayStatusResponses maps entity day statuses to HTTP responses.
func ToMyDayStatusResponses(days []*entity.MyDayStatus) []MyDayStatusResponse {
	out := make([]MyDayStatusResponse, 0, len(days))
	for _, d := range days {
		r := MyDayStatusResponse{Date: d.Date.UTC().Format(DateLayout), DayStatus: d.DayStatus, HalfDay: d.HalfDay, TotalHours: d.TotalHours}
		if d.HolidayName != "" {
			n := d.HolidayName
			r.HolidayName = &n
		}
		if d.LeaveTypeName != "" {
			n := d.LeaveTypeName
			r.LeaveTypeName = &n
		}
		out = append(out, r)
	}
	return out
}
