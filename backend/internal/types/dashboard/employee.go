package dashboard

// Day kinds for EmployeeToday.Kind.
const (
	DayKindWorking   = "working_day"
	DayKindHoliday   = "holiday"
	DayKindWeeklyOff = "weekly_off"
	// DayKindLeave is a working day covered by approved full-day leave.
	DayKindLeave = "leave"
)

// Employee dashboard list sizes.
const (
	EmployeeRecentLeaveLimit = 3
	EmployeeHolidayLimit     = 5
	// EmployeeWeekDays is the length of the hours window ending today.
	EmployeeWeekDays = 7
)

// EmployeeToday describes today's day type.
type EmployeeToday struct {
	// Date is today (UTC) as YYYY-MM-DD.
	Date string `json:"date" example:"2026-10-30"`
	// Kind is one of working_day, holiday, weekly_off, leave.
	Kind string `json:"kind" example:"working_day" enums:"working_day,holiday,weekly_off,leave"`
	// HolidayName is set only when Kind is holiday.
	HolidayName string `json:"holiday_name,omitempty"`
}

// EmployeeGreeting is the dashboard header data.
type EmployeeGreeting struct {
	FirstName      string        `json:"first_name"`
	WelcomeMessage string        `json:"welcome_message"`
	Today          EmployeeToday `json:"today"`
}

// EmployeeStats are the KPI tiles.
type EmployeeStats struct {
	PendingLeave int `json:"pending_leave"`
	// UpcomingLeaveDays totals the remaining days of approved requests that have not ended: counted from today (or the start date, if later) through the end date.
	UpcomingLeaveDays float64 `json:"upcoming_leave_days"`
	// AllocationPercent is today's total on live projects.
	AllocationPercent int `json:"allocation_percent"`
	// HoursThisWeek totals the hours logged over the last 7 days (a rolling window ending today, not the calendar week). The name is kept for API compatibility.
	HoursThisWeek float64 `json:"hours_this_week"`
	// PendingApprovals is the caller's pending team-approval count; null unless the caller is an approver.
	PendingApprovals *int `json:"pending_approvals"`
}

// EmployeeLeaveItem is one leave request summary.
type EmployeeLeaveItem struct {
	ID            string  `json:"id"`
	LeaveTypeName string  `json:"leave_type_name"`
	LeaveTypeCode string  `json:"leave_type_code"`
	StartDate     string  `json:"start_date" example:"2026-11-02"`
	EndDate       string  `json:"end_date" example:"2026-11-03"`
	DayPart       string  `json:"day_part"`
	Days          float64 `json:"days"`
	Status        string  `json:"status"`
}

// EmployeeLeaveOverview is the leave card.
type EmployeeLeaveOverview struct {
	// NextApproved is the next approved leave that has not ended; null when none.
	NextApproved *EmployeeLeaveItem  `json:"next_approved"`
	Recent       []EmployeeLeaveItem `json:"recent"`
}

// EmployeeHoliday is an upcoming holiday.
type EmployeeHoliday struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Date          string `json:"date" example:"2026-12-25"`
	CategoryName  string `json:"category_name"`
	CategoryColor string `json:"category_color"`
	DaysUntil     int    `json:"days_until"`
}

// EmployeeProject is an active allocation.
type EmployeeProject struct {
	ProjectID         string  `json:"project_id"`
	ProjectName       string  `json:"project_name"`
	ProjectCode       string  `json:"project_code"`
	RoleOnProject     *string `json:"role_on_project"`
	AllocationPercent int     `json:"allocation_percent"`
	EndDate           *string `json:"end_date" example:"2026-12-31"`
}

// EmployeeWeekDay is one day of the rolling last-7-days hours window.
type EmployeeWeekDay struct {
	Date  string  `json:"date" example:"2026-10-30"`
	Hours float64 `json:"hours"`
	// IsWorkingDay is false on weekly off days and holidays.
	IsWorkingDay bool `json:"is_working_day"`
	// Kind is one of working_day, holiday, weekly_off, leave (approved full-day leave on a working day; half-day leave stays working_day).
	Kind string `json:"kind" enums:"working_day,holiday,weekly_off,leave"`
}

// EmployeeWeek is the last-7-days hours card (rolling window ending today, not the calendar week).
type EmployeeWeek struct {
	Days []EmployeeWeekDay `json:"days"`
	// MissingDays counts past working days (today excluded) with no hours logged and no approved leave (full or half day).
	MissingDays int `json:"missing_days"`
}

// EmployeeDashboardResponse is the employee home payload for the caller only.
type EmployeeDashboardResponse struct {
	Greeting EmployeeGreeting      `json:"greeting"`
	Stats    EmployeeStats         `json:"stats"`
	Leave    EmployeeLeaveOverview `json:"leave"`
	Holidays []EmployeeHoliday     `json:"holidays"`
	Projects []EmployeeProject     `json:"projects"`
	Week     EmployeeWeek          `json:"week"`
}
