package employee

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// Actor is the authenticated caller, resolved by the handler from the auth
// context and passed explicitly to usecases.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	Roles    []string
}

// HasRole reports whether the actor holds the role.
func (a Actor) HasRole(role string) bool {
	for _, r := range a.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// UpdateEmployeeInput is the body of PUT/PATCH /employees/{id}. Absent fields
// are left unchanged; an explicit null (or empty string) clears optional text,
// date and reference fields. Tenant identity is never part of the payload.
type UpdateEmployeeInput struct {
	DepartmentID  Optional[uuid.UUID] `json:"department_id" swaggertype:"string"`
	PositionID    Optional[uuid.UUID] `json:"position_id" swaggertype:"string"`
	ManagerID     Optional[uuid.UUID] `json:"manager_id" swaggertype:"string"`
	EffectiveDate Optional[Date]      `json:"effective_date" swaggertype:"string" format:"date"`
	ChangeReason  Optional[string]    `json:"change_reason" swaggertype:"string"`

	EmployeeCode     Optional[string] `json:"employee_code" swaggertype:"string"`
	EmploymentType   Optional[string] `json:"employment_type" swaggertype:"string" enums:"full_time,part_time,contract,intern,consultant"`
	EmploymentStatus Optional[string] `json:"employment_status" swaggertype:"string" enums:"active,probation,on_leave,notice_period,resigned,terminated"`
	JoiningDate      Optional[Date]   `json:"joining_date" swaggertype:"string" format:"date"`
	ProbationEndDate Optional[Date]   `json:"probation_end_date" swaggertype:"string" format:"date"`
	ConfirmationDate Optional[Date]   `json:"confirmation_date" swaggertype:"string" format:"date"`
	ExitDate         Optional[Date]   `json:"exit_date" swaggertype:"string" format:"date"`
	WorkLocation     Optional[string] `json:"work_location" swaggertype:"string"`

	SelfProfileFields
}

// SelfProfileFields are the personal and address fields an employee may fill in
// themselves (also editable by an admin).
type SelfProfileFields struct {
	// FirstName and LastName live on the users row. When provided they must be
	// non-empty after trimming and at most 100 characters.
	FirstName Optional[string] `json:"first_name" swaggertype:"string"`
	LastName  Optional[string] `json:"last_name" swaggertype:"string"`

	Gender        Optional[string] `json:"gender" swaggertype:"string" enums:"male,female,other,prefer_not_to_say"`
	DateOfBirth   Optional[Date]   `json:"date_of_birth" swaggertype:"string" format:"date"`
	BloodGroup    Optional[string] `json:"blood_group" swaggertype:"string" enums:"A+,A-,B+,B-,AB+,AB-,O+,O-"`
	MaritalStatus Optional[string] `json:"marital_status" swaggertype:"string" enums:"single,married,divorced,widowed,other"`
	Nationality   Optional[string] `json:"nationality" swaggertype:"string"`
	Phone         Optional[string] `json:"phone" swaggertype:"string"`
	PersonalEmail Optional[string] `json:"personal_email" swaggertype:"string"`
	AddressLine1  Optional[string] `json:"address_line1" swaggertype:"string"`
	AddressLine2  Optional[string] `json:"address_line2" swaggertype:"string"`
	City          Optional[string] `json:"city" swaggertype:"string"`
	State         Optional[string] `json:"state" swaggertype:"string"`
	Country       Optional[string] `json:"country" swaggertype:"string"`
	Pincode       Optional[string] `json:"pincode" swaggertype:"string"`
}

// UpdateMyProfileInput is the body of PATCH /me/profile. It deliberately has no
// employment fields: those are read-only for the employee.
type UpdateMyProfileInput struct {
	SelfProfileFields
}

// EmergencyContactInput is the body for creating or replacing an emergency contact.
type EmergencyContactInput struct {
	Name           string  `json:"name" binding:"required,max=100"`
	Relationship   string  `json:"relationship" binding:"required,max=50"`
	Phone          string  `json:"phone" binding:"required,max=30"`
	AlternatePhone *string `json:"alternate_phone,omitempty" binding:"omitempty,max=30"`
	Email          *string `json:"email,omitempty" binding:"omitempty,max=200"`
	// IsPrimary makes this the single primary contact (any previous primary is cleared).
	IsPrimary bool `json:"is_primary"`
}

// ChangeRoleInput is the body of PUT /employees/{id}/role.
type ChangeRoleInput struct {
	Role string `json:"role" binding:"required,oneof=employee admin"`
}

// ListEmployeesRequest holds the raw filter query parameters of GET /employees
// (pagination is parsed separately by the handler). Validation lives in ToFilter
// so error messages stay stable; tenant identity is never a query parameter.
type ListEmployeesRequest struct {
	Include          string `form:"include" enums:"allocation"`
	Search           string `form:"search"`
	Status           string `form:"status" enums:"active,inactive"`
	EmploymentStatus string `form:"employment_status"`
	DepartmentID     string `form:"department_id"`
	PositionID       string `form:"position_id"`
}

// ToFilter validates the request and converts it to the domain filter. The
// returned error message is safe to show to the client.
func (r ListEmployeesRequest) ToFilter() (entity.EmployeeFilter, error) {
	f := entity.EmployeeFilter{Search: strings.TrimSpace(r.Search), EmploymentStatus: r.EmploymentStatus}
	switch r.Status {
	case "":
	case "active":
		t := true
		f.IsActive = &t
	case "inactive":
		t := false
		f.IsActive = &t
	default:
		return f, errors.New("status must be active or inactive")
	}
	switch r.Include {
	case "":
	case "allocation":
		f.IncludeAllocation = true
	default:
		return f, errors.New("include must be allocation")
	}
	var err error
	if f.DepartmentID, err = optionalUUID("department_id", r.DepartmentID); err != nil {
		return f, err
	}
	if f.PositionID, err = optionalUUID("position_id", r.PositionID); err != nil {
		return f, err
	}
	return f, nil
}

func optionalUUID(key, raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, errors.New(key + " must be a UUID")
	}
	return &id, nil
}

// ListEmployeesQuery holds list parameters.
type ListEmployeesQuery struct {
	Page     int
	PageSize int
	Filter   entity.EmployeeFilter
}

// ListEmployeesResult is a page of employees.
type ListEmployeesResult struct {
	Employees []*entity.EmployeeSummary
	Total     int64
	Page      int
	PageSize  int
}

// EmployeeDetail is the full view of one employee.
type EmployeeDetail struct {
	Summary  *entity.EmployeeSummary
	Profile  *entity.EmployeeProfile // nil when no profile row exists yet
	Contacts []*entity.EmergencyContact
}

// MyProfile is the employee's own profile view, with completion state.
type MyProfile struct {
	FirstName     string
	LastName      string
	Profile       *entity.EmployeeProfile // nil when no profile row exists yet
	Contacts      []*entity.EmergencyContact
	Completed     bool
	MissingFields []string // includes "emergency_contact" when there is none
	// Read-only employment placement, resolved from the user's department/position.
	DepartmentID   *uuid.UUID
	DepartmentName string
	PositionID     *uuid.UUID
	PositionName   string
}

// Me is the result of GET /me.
type Me struct {
	User             *entity.User
	Roles            []string
	OrganizationID   uuid.UUID
	OrganizationName string
	FirstLogin       bool
	ProfileCompleted bool
	// ProfileRequired is true for users subject to profile gating (non-admins).
	ProfileRequired bool
}

// ---- Responses ----

// EmployeeSummaryResponse is one row of the employee list.
type EmployeeSummaryResponse struct {
	ID               uuid.UUID  `json:"id"`
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
	JoiningDate      *string    `json:"joining_date,omitempty"`
	WorkLocation     *string    `json:"work_location,omitempty"`
	ManagerID        *uuid.UUID `json:"manager_id,omitempty"`
	Phone            *string    `json:"phone,omitempty"`
	ProfileCompleted bool       `json:"profile_completed"`
	CreatedAt        time.Time  `json:"created_at"`
	// AllocatedPercent is only present when requested with include=allocation:
	// the employee's total allocation today on live projects.
	AllocatedPercent *int `json:"allocated_percent,omitempty"`
}

// ToSummaryResponse maps the read model to its response.
func ToSummaryResponse(s *entity.EmployeeSummary) EmployeeSummaryResponse {
	roles := s.Roles
	if roles == nil {
		roles = []string{}
	}
	return EmployeeSummaryResponse{
		ID: s.UserID, Email: s.Email, FirstName: s.FirstName, LastName: s.LastName,
		IsActive: s.IsActive, LastLoginAt: s.LastLoginAt,
		DepartmentID: s.DepartmentID, DepartmentName: s.DepartmentName,
		PositionID: s.PositionID, PositionName: s.PositionName, Roles: roles,
		EmployeeCode: s.EmployeeCode, EmploymentType: s.EmploymentType, EmploymentStatus: s.EmploymentStatus,
		JoiningDate: FormatDate(s.JoiningDate), WorkLocation: s.WorkLocation, ManagerID: s.ManagerID,
		Phone: s.Phone, ProfileCompleted: s.ProfileCompleted, CreatedAt: s.CreatedAt,
		AllocatedPercent: s.AllocatedPercent,
	}
}

// ProfileResponse is the profile of an employee.
type ProfileResponse struct {
	EmployeeCode       *string    `json:"employee_code,omitempty"`
	Gender             *string    `json:"gender,omitempty"`
	DateOfBirth        *string    `json:"date_of_birth,omitempty"`
	BloodGroup         *string    `json:"blood_group,omitempty"`
	MaritalStatus      *string    `json:"marital_status,omitempty"`
	Nationality        *string    `json:"nationality,omitempty"`
	Phone              *string    `json:"phone,omitempty"`
	PersonalEmail      *string    `json:"personal_email,omitempty"`
	JoiningDate        *string    `json:"joining_date,omitempty"`
	EmploymentType     *string    `json:"employment_type,omitempty"`
	EmploymentStatus   string     `json:"employment_status"`
	ProbationEndDate   *string    `json:"probation_end_date,omitempty"`
	ConfirmationDate   *string    `json:"confirmation_date,omitempty"`
	ExitDate           *string    `json:"exit_date,omitempty"`
	WorkLocation       *string    `json:"work_location,omitempty"`
	ManagerID          *uuid.UUID `json:"manager_id,omitempty"`
	AddressLine1       *string    `json:"address_line1,omitempty"`
	AddressLine2       *string    `json:"address_line2,omitempty"`
	City               *string    `json:"city,omitempty"`
	State              *string    `json:"state,omitempty"`
	Country            *string    `json:"country,omitempty"`
	Pincode            *string    `json:"pincode,omitempty"`
	ProfileCompletedAt *time.Time `json:"profile_completed_at,omitempty"`
}

// ToProfileResponse maps a profile to its response; nil yields nil.
func ToProfileResponse(p *entity.EmployeeProfile) *ProfileResponse {
	if p == nil {
		return nil
	}
	return &ProfileResponse{
		EmployeeCode: p.EmployeeCode, Gender: p.Gender, DateOfBirth: FormatDate(p.DateOfBirth),
		BloodGroup: p.BloodGroup, MaritalStatus: p.MaritalStatus, Nationality: p.Nationality,
		Phone: p.Phone, PersonalEmail: p.PersonalEmail, JoiningDate: FormatDate(p.JoiningDate),
		EmploymentType: p.EmploymentType, EmploymentStatus: p.EmploymentStatus,
		ProbationEndDate: FormatDate(p.ProbationEndDate), ConfirmationDate: FormatDate(p.ConfirmationDate),
		ExitDate: FormatDate(p.ExitDate), WorkLocation: p.WorkLocation, ManagerID: p.ManagerID,
		AddressLine1: p.AddressLine1, AddressLine2: p.AddressLine2, City: p.City, State: p.State,
		Country: p.Country, Pincode: p.Pincode, ProfileCompletedAt: p.ProfileCompletedAt,
	}
}

// EmergencyContactResponse is one emergency contact.
type EmergencyContactResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Relationship   string    `json:"relationship"`
	Phone          string    `json:"phone"`
	AlternatePhone *string   `json:"alternate_phone,omitempty"`
	Email          *string   `json:"email,omitempty"`
	IsPrimary      bool      `json:"is_primary"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ToContactResponse maps a contact to its response.
func ToContactResponse(c *entity.EmergencyContact) EmergencyContactResponse {
	return EmergencyContactResponse{
		ID: c.ID, Name: c.Name, Relationship: c.Relationship, Phone: c.Phone,
		AlternatePhone: c.AlternatePhone, Email: c.Email, IsPrimary: c.IsPrimary,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

// ToContactResponses maps a slice (never nil).
func ToContactResponses(cs []*entity.EmergencyContact) []EmergencyContactResponse {
	out := make([]EmergencyContactResponse, 0, len(cs))
	for _, c := range cs {
		out = append(out, ToContactResponse(c))
	}
	return out
}

// EmployeeDetailResponse is the full employee view.
type EmployeeDetailResponse struct {
	EmployeeSummaryResponse
	Profile  *ProfileResponse           `json:"profile"`
	Contacts []EmergencyContactResponse `json:"emergency_contacts"`
}

// ToDetailResponse maps an EmployeeDetail.
func ToDetailResponse(d *EmployeeDetail) EmployeeDetailResponse {
	return EmployeeDetailResponse{
		EmployeeSummaryResponse: ToSummaryResponse(d.Summary),
		Profile:                 ToProfileResponse(d.Profile),
		Contacts:                ToContactResponses(d.Contacts),
	}
}

// AssignmentResponse is one department/position stint.
type AssignmentResponse struct {
	ID            uuid.UUID  `json:"id"`
	DepartmentID  *uuid.UUID `json:"department_id,omitempty"`
	PositionID    *uuid.UUID `json:"position_id,omitempty"`
	EffectiveFrom string     `json:"effective_from"`
	EffectiveTo   *string    `json:"effective_to,omitempty"`
	ChangeReason  *string    `json:"change_reason,omitempty"`
	IsCurrent     bool       `json:"is_current"`
}

// ToAssignmentResponses maps assignment history (newest first, as given).
func ToAssignmentResponses(as []*entity.UserAssignment) []AssignmentResponse {
	out := make([]AssignmentResponse, 0, len(as))
	for _, a := range as {
		out = append(out, AssignmentResponse{
			ID: a.ID, DepartmentID: a.DepartmentID, PositionID: a.PositionID,
			EffectiveFrom: a.EffectiveFrom.UTC().Format(DateLayout), EffectiveTo: FormatDate(a.EffectiveTo),
			ChangeReason: a.ChangeReason, IsCurrent: a.EffectiveTo == nil,
		})
	}
	return out
}

// MyProfileResponse is GET /me/profile.
type MyProfileResponse struct {
	FirstName        string                     `json:"first_name"`
	LastName         string                     `json:"last_name"`
	Profile          *ProfileResponse           `json:"profile"`
	EmergencyContact []EmergencyContactResponse `json:"emergency_contacts"`
	ProfileCompleted bool                       `json:"profile_completed"`
	MissingFields    []string                   `json:"missing_fields"`
	// Read-only employment placement; omitted when unassigned.
	DepartmentID   *uuid.UUID `json:"department_id,omitempty"`
	DepartmentName string     `json:"department_name,omitempty"`
	PositionID     *uuid.UUID `json:"position_id,omitempty"`
	PositionName   string     `json:"position_name,omitempty"`
}

// ToMyProfileResponse maps MyProfile.
func ToMyProfileResponse(m *MyProfile) MyProfileResponse {
	missing := m.MissingFields
	if missing == nil {
		missing = []string{}
	}
	return MyProfileResponse{
		FirstName: m.FirstName, LastName: m.LastName,
		Profile: ToProfileResponse(m.Profile), EmergencyContact: ToContactResponses(m.Contacts),
		ProfileCompleted: m.Completed, MissingFields: missing,
		DepartmentID: m.DepartmentID, DepartmentName: m.DepartmentName,
		PositionID: m.PositionID, PositionName: m.PositionName,
	}
}

// MeResponse is GET /me.
type MeResponse struct {
	ID               uuid.UUID `json:"id"`
	Email            string    `json:"email"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Roles            []string  `json:"roles"`
	OrganizationID   uuid.UUID `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	FirstLogin       bool      `json:"first_login"`
	ProfileCompleted bool      `json:"profile_completed"`
	ProfileRequired  bool      `json:"profile_required"`
}

// ToMeResponse maps Me.
func ToMeResponse(m *Me) MeResponse {
	roles := m.Roles
	if roles == nil {
		roles = []string{}
	}
	return MeResponse{
		ID: m.User.ID, Email: m.User.Email, FirstName: m.User.FirstName, LastName: m.User.LastName,
		Roles: roles, OrganizationID: m.OrganizationID, OrganizationName: m.OrganizationName,
		FirstLogin: m.FirstLogin, ProfileCompleted: m.ProfileCompleted, ProfileRequired: m.ProfileRequired,
	}
}
