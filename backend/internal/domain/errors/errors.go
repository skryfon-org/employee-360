package errors

import "errors"

var (
	// Generic errors
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInternal      = errors.New("internal error")

	// Tenant errors
	ErrTenantNotFound        = errors.New("tenant not found")
	ErrEmailDomainNotAllowed = errors.New("email domain is not registered for this organization")
	ErrDomainAlreadyExists   = errors.New("domain already registered")
	ErrDomainNotFound        = errors.New("tenant domain not found")
	ErrInvalidDomain         = errors.New("invalid domain name")
	ErrInvalidTenantName     = errors.New("invalid tenant name")
	// ErrLastDomain is returned when removing a tenant's only live domain.
	ErrLastDomain = errors.New("cannot remove the last domain of a tenant")

	// ErrDomainInUse is returned when removing or changing a tenant domain
	// that live (not soft-deleted) users of the tenant still sign in with.
	ErrDomainInUse = errors.New("domain is still used by users of the tenant")

	// User errors
	ErrUserNotFound       = errors.New("user not found")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrEmailAlreadyExists = errors.New("email already registered in tenant")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidPassword    = errors.New("password must be between 8 and 72 bytes")

	// Role errors
	ErrRoleNotFound     = errors.New("role not found")
	ErrUserRoleNotFound = errors.New("user role not found")

	// Token errors
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrTokenExpired = errors.New("token has expired")
	ErrTokenRevoked = errors.New("token has been revoked")
	// ErrTokenAlreadyRevoked is returned by RefreshTokenRepository.Revoke when
	// the conditional revoke matched no row, i.e. the token was already revoked
	// (a concurrent or repeated use of the same refresh token).
	ErrTokenAlreadyRevoked = errors.New("refresh token already revoked")
	// ErrTokenAlreadyUsed is returned by PasswordResetRepository.MarkAsUsed when
	// the conditional update matched no row, i.e. the token was already consumed.
	ErrTokenAlreadyUsed = errors.New("password reset token already used")

	// Invitation errors
	ErrInvitationNotFound   = errors.New("invitation not found")
	ErrInvitationNotPending = errors.New("invitation is no longer pending")
	// Distinct token-state errors returned by validate/accept when the token
	// hash matches a row but the invitation cannot be used.
	ErrInvitationExpired       = errors.New("invitation has expired")
	ErrInvitationRevoked       = errors.New("invitation has been revoked")
	ErrInvitationAccepted      = errors.New("invitation has already been accepted")
	ErrInvalidRole             = errors.New("role cannot be assigned by invitation")
	ErrInvalidEmail            = errors.New("invalid email address")
	ErrInvalidInvitationFilter = errors.New("invalid invitation list filter")
	ErrDepartmentNotFound      = errors.New("department not found")
	ErrDepartmentInactive      = errors.New("department is inactive")
	ErrDepartmentNameTaken     = errors.New("department name already exists in this tenant")
	ErrDepartmentInUse         = errors.New("department is currently assigned to users or invitations")
	ErrPositionNotFound        = errors.New("position not found")
	ErrPositionInactive        = errors.New("position is inactive")
	ErrPositionNameTaken       = errors.New("position name already exists in this tenant")
	ErrPositionInUse           = errors.New("position is currently assigned to users or invitations")
	ErrAuditLogNotFound        = errors.New("audit log not found")
	ErrInvalidAuditLogFilter   = errors.New("invalid audit log filter")

	// Holiday errors
	ErrHolidayNotFound                            = errors.New("holiday not found")
	ErrHolidayDuplicate                           = errors.New("holiday already exists for this date and name")
	ErrHolidayNameRequired                        = errors.New("holiday name is required")
	ErrHolidayNameTooLong                         = errors.New("holiday name must not exceed 150 characters")
	ErrHolidayDateInvalid                         = errors.New("holiday date is invalid")
	ErrHolidayYearInvalid                         = errors.New("holiday year must be between 2000 and 2100")
	ErrHolidayMonthInvalid                        = errors.New("holiday month must be between 1 and 12")
	ErrHolidayCategoryNotFound                    = errors.New("holiday category not found")
	ErrHolidayCategoryInactive                    = errors.New("holiday category is inactive")
	ErrHolidayCategoryDuplicate                   = errors.New("holiday category already exists with this name")
	ErrHolidayCategoryInUse                       = errors.New("holiday category is in use by holidays")
	ErrHolidayCategoryNameRequired                = errors.New("holiday category name is required")
	ErrHolidayCategoryNameTooLong                 = errors.New("holiday category name must not exceed 100 characters")
	ErrHolidayCategoryColorInvalid                = errors.New("holiday category color must be a valid hex color code (e.g. #RRGGBB)")
	ErrHolidayCategoryDescTooLong                 = errors.New("holiday category description must not exceed 500 characters")
	ErrHolidayCalendarSettingsNotFound            = errors.New("holiday calendar settings not found")
	ErrHolidayCalendarSettingsInvalidDefaultView  = errors.New("default view must be either 'calendar' or 'list'")
	ErrHolidayCalendarSettingsInvalidDefaultYear  = errors.New("default year must be between 2000 and 2100")
	ErrHolidayCalendarSettingsInvalidWeekStartDay = errors.New("week start day must be between 0 and 6")

	// Employee management errors
	ErrEmployeeProfileNotFound  = errors.New("employee profile not found")
	ErrEmployeeProfileExists    = errors.New("employee profile already exists for this user; retry")
	ErrEmployeeCodeTaken        = errors.New("employee code already in use")
	ErrEmployeeCodeRequired     = errors.New("employee code must be assigned before creating a profile")
	ErrEmergencyContactNotFound = errors.New("emergency contact not found")
	ErrInvalidEmployeeData      = errors.New("invalid employee data")
	ErrManagerInvalid           = errors.New("manager must be another active user of this organization")
	ErrAssignmentDateOrder      = errors.New("effective date is before the start of the current assignment")
	ErrCannotActOnSelf          = errors.New("you cannot perform this action on your own account")
	ErrCannotActOnSuperAdmin    = errors.New("only a super admin can act on a super admin")
	ErrLastActiveAdmin          = errors.New("cannot remove the last active admin of the organization")
	ErrUserNotOnboarded         = errors.New("user has not completed onboarding and cannot be activated")
	ErrRoleChangeNotAllowed     = errors.New("role change is not allowed")
	ErrProfileIncomplete        = errors.New("profile is incomplete")
)

// Organization settings errors
var (
	ErrOrganizationSettingsNotFound = errors.New("organization settings not found")
	// ErrInvalidOrganizationSettings is matched (errors.Is) by every client-correctable
	// organization settings validation failure; the HTTP layer maps it to 400.
	ErrInvalidOrganizationSettings = errors.New("invalid organization settings")
)

// Project and allocation errors.
var (
	ErrProjectNotFound        = errors.New("project not found")
	ErrProjectCodeTaken       = errors.New("project code already exists in this tenant")
	ErrProjectArchived        = errors.New("project is archived and cannot receive new allocations")
	ErrAllocationNotFound     = errors.New("allocation not found")
	ErrAllocationOverlap      = errors.New("user already has an overlapping allocation on this project")
	ErrAllocationUserInactive = errors.New("user is inactive and cannot be allocated")
	// ErrInvalidProjectData is matched (errors.Is) by every client-correctable
	// project/allocation validation failure; its message is safe to show.
	ErrInvalidProjectData = errors.New("invalid project data")
)

// Status log errors.
var (
	ErrStatusLogNotFound          = errors.New("status log entry not found")
	ErrLogWindowClosed            = errors.New("status log window is closed for this date")
	ErrFutureDate                 = errors.New("cannot log status for future date")
	ErrDailyHoursExceeded         = errors.New("daily total hours cannot exceed 24")
	ErrNoActiveAllocation         = errors.New("project entries require an active allocation on log date")
	ErrProjectOrCategoryRequired  = errors.New("exactly one of project or activity category must be set")
	ErrInvalidActivityCategory    = errors.New("invalid activity category")
	ErrInvalidStatusLogHours      = errors.New("hours must be greater than 0 and at most 24")
	ErrInvalidStatusLogDate       = errors.New("invalid status log date")
	ErrStatusLogDateRangeExceeded = errors.New("status log date range cannot exceed 366 days")
)

// Leave errors.
var (
	ErrLeaveTypeNotFound    = errors.New("leave type not found")
	ErrLeaveTypeCodeTaken   = errors.New("leave type code already exists in this tenant")
	ErrLeaveTypeInactive    = errors.New("leave type is inactive")
	ErrLeaveRequestNotFound = errors.New("leave request not found")
	ErrLeaveOverlap         = errors.New("leave request overlaps an existing pending or approved request")
	ErrLeaveStartTooOld     = errors.New("start date cannot be in the past")
	ErrInvalidLeaveApprover = errors.New("approver must be your manager or a tenant admin, and not yourself")
	ErrLeaveNotPending      = errors.New("leave request is not pending")
	ErrLeaveNotCancellable  = errors.New("leave request can no longer be cancelled")
	ErrLeaveSelfDecision    = errors.New("you cannot approve or reject your own leave request")
	// ErrInvalidLeaveData is matched (errors.Is) by every client-correctable
	// leave validation failure; its message is safe to show.
	ErrInvalidLeaveData = errors.New("invalid leave data")
)
