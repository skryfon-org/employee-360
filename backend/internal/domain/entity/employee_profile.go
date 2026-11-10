package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Enumerated profile values (mirrored by CHECK constraints in migration 000024).
var (
	Genders            = []string{"male", "female", "other", "prefer_not_to_say"}
	BloodGroups        = []string{"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"}
	MaritalStatuses    = []string{"single", "married", "divorced", "widowed", "other"}
	EmploymentTypes    = []string{"full_time", "part_time", "contract", "intern", "consultant"}
	EmploymentStatuses = []string{"active", "probation", "on_leave", "notice_period", "resigned", "terminated"}
)

// EmploymentStatusActive is the default employment status.
const EmploymentStatusActive = "active"

// EmployeeProfile holds the personal, employment and address details of a user
// (1:1 with users). Date-only fields are carried as time.Time at UTC midnight.
type EmployeeProfile struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	UserID   uuid.UUID `json:"user_id"`

	EmployeeCode  *string    `json:"employee_code,omitempty"`
	Gender        *string    `json:"gender,omitempty"`
	DateOfBirth   *time.Time `json:"date_of_birth,omitempty"`
	BloodGroup    *string    `json:"blood_group,omitempty"`
	MaritalStatus *string    `json:"marital_status,omitempty"`
	Nationality   *string    `json:"nationality,omitempty"`
	Phone         *string    `json:"phone,omitempty"`
	PersonalEmail *string    `json:"personal_email,omitempty"`

	JoiningDate      *time.Time `json:"joining_date,omitempty"`
	EmploymentType   *string    `json:"employment_type,omitempty"`
	EmploymentStatus string     `json:"employment_status"`
	ProbationEndDate *time.Time `json:"probation_end_date,omitempty"`
	ConfirmationDate *time.Time `json:"confirmation_date,omitempty"`
	ExitDate         *time.Time `json:"exit_date,omitempty"`
	WorkLocation     *string    `json:"work_location,omitempty"`
	ManagerID        *uuid.UUID `json:"manager_id,omitempty"`

	AddressLine1 *string `json:"address_line1,omitempty"`
	AddressLine2 *string `json:"address_line2,omitempty"`
	City         *string `json:"city,omitempty"`
	State        *string `json:"state,omitempty"`
	Country      *string `json:"country,omitempty"`
	Pincode      *string `json:"pincode,omitempty"`

	ProfileCompletedAt *time.Time `json:"profile_completed_at,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

func filled(s *string) bool { return s != nil && strings.TrimSpace(*s) != "" }

// MissingRequiredFields returns the names of the mandatory self-filled fields
// that are not yet present: phone, date_of_birth, gender, address_line1, city,
// country. (The "at least one emergency contact" rule is checked separately.)
func (p *EmployeeProfile) MissingRequiredFields() []string {
	var missing []string
	if p == nil {
		return []string{"phone", "date_of_birth", "gender", "address_line1", "city", "country"}
	}
	if !filled(p.Phone) {
		missing = append(missing, "phone")
	}
	if p.DateOfBirth == nil {
		missing = append(missing, "date_of_birth")
	}
	if !filled(p.Gender) {
		missing = append(missing, "gender")
	}
	if !filled(p.AddressLine1) {
		missing = append(missing, "address_line1")
	}
	if !filled(p.City) {
		missing = append(missing, "city")
	}
	if !filled(p.Country) {
		missing = append(missing, "country")
	}
	return missing
}

// EmergencyContact is a person to contact in an emergency for an employee.
type EmergencyContact struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	UserID         uuid.UUID `json:"user_id"`
	Name           string    `json:"name"`
	Relationship   string    `json:"relationship"`
	Phone          string    `json:"phone"`
	AlternatePhone *string   `json:"alternate_phone,omitempty"`
	Email          *string   `json:"email,omitempty"`
	IsPrimary      bool      `json:"is_primary"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

// UserAssignment is one department/position stint of a user. EffectiveTo nil
// means the stint is current (open).
type UserAssignment struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	UserID        uuid.UUID  `json:"user_id"`
	DepartmentID  *uuid.UUID `json:"department_id,omitempty"`
	PositionID    *uuid.UUID `json:"position_id,omitempty"`
	EffectiveFrom time.Time  `json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty"`
	ChangeReason  *string    `json:"change_reason,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

// EmployeeSummary is the read model of one row of the employee directory: user
// identity joined with department/position names, roles and a profile summary.
type EmployeeSummary struct {
	UserID           uuid.UUID  `json:"user_id"`
	Email            string     `json:"email"`
	FirstName        string     `json:"first_name"`
	LastName         string     `json:"last_name"`
	IsActive         bool       `json:"is_active"`
	LastLoginAt      *time.Time `json:"last_login_at,omitempty"`
	DepartmentID     *uuid.UUID `json:"department_id,omitempty"`
	DepartmentName   *string    `json:"department_name,omitempty"`
	PositionID       *uuid.UUID `json:"position_id,omitempty"`
	PositionName     *string    `json:"position_name,omitempty"`
	Roles            []string   `json:"roles"`
	EmployeeCode     *string    `json:"employee_code,omitempty"`
	EmploymentType   *string    `json:"employment_type,omitempty"`
	EmploymentStatus *string    `json:"employment_status,omitempty"`
	JoiningDate      *time.Time `json:"joining_date,omitempty"`
	WorkLocation     *string    `json:"work_location,omitempty"`
	ManagerID        *uuid.UUID `json:"manager_id,omitempty"`
	Phone            *string    `json:"phone,omitempty"`
	ProfileCompleted bool       `json:"profile_completed"`
	CreatedAt        time.Time  `json:"created_at"`
	// AllocatedPercent is set only when the list was requested with
	// IncludeAllocation: the sum of allocations active on the as-of date on
	// live projects.
	AllocatedPercent *int `json:"-"`
}

// EmployeeFilter narrows the employee directory listing. All fields are optional.
type EmployeeFilter struct {
	Search           string
	IsActive         *bool
	DepartmentID     *uuid.UUID
	PositionID       *uuid.UUID
	EmploymentStatus string
	// IncludeAllocation makes List populate EmployeeSummary.AllocatedPercent
	// as of AllocationAsOf (picker use; skipped otherwise).
	IncludeAllocation bool
	AllocationAsOf    time.Time
}
