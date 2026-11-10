// Package audit centralizes the audit-log action and entity-type identifiers.
// Values are persisted in audit_logs and matched by the admin viewer, so they
// must never change. Actions follow the "entity.verb" convention.
package audit

// Entity types.
const (
	EntityDepartment              = "department"
	EntityPosition                = "position"
	EntityInvitation              = "user_invitation"
	EntityTenant                  = "tenant"
	EntityTenantDomain            = "tenant_domain"
	EntityHoliday                 = "holiday"
	EntityHolidayCategory         = "holiday_category"
	EntityHolidayCalendarSettings = "holiday_calendar_settings"
	EntityEmployee                = "employee"
	EntityOrganizationSettings    = "organization_settings"
	EntityProject                 = "project"
	EntityProjectAllocation       = "project_allocation"
	EntityLeaveType               = "leave_type"
	EntityLeaveRequest            = "leave_request"
)

// Actions.
const (
	ActionDepartmentCreate     = "department.create"
	ActionDepartmentUpdate     = "department.update"
	ActionDepartmentDelete     = "department.delete"
	ActionDepartmentActivate   = "department.activate"
	ActionDepartmentDeactivate = "department.deactivate"

	ActionPositionCreate     = "position.create"
	ActionPositionUpdate     = "position.update"
	ActionPositionDelete     = "position.delete"
	ActionPositionActivate   = "position.activate"
	ActionPositionDeactivate = "position.deactivate"

	ActionInvitationInvite = "invitation.invite"
	ActionInvitationResend = "invitation.resend"
	ActionInvitationRevoke = "invitation.revoke"

	ActionTenantRename       = "tenant.rename"
	ActionTenantDomainAdd    = "tenant.domain.add"
	ActionTenantDomainUpdate = "tenant.domain.update"
	ActionTenantDomainRemove = "tenant.domain.remove"

	ActionHolidayCreate = "holiday.create"
	ActionHolidayUpdate = "holiday.update"
	ActionHolidayDelete = "holiday.delete"

	ActionHolidayCategoryCreate = "holiday_category.create"
	ActionHolidayCategoryUpdate = "holiday_category.update"
	ActionHolidayCategoryDelete = "holiday_category.delete"

	ActionHolidayCalendarSettingsUpdate = "holiday_calendar_settings.update"

	ActionOrganizationSettingsUpdate = "organization_settings.update"

	ActionEmployeeUpdate            = "employee.update"
	ActionEmployeeAssignmentChange  = "employee.assignment_change"
	ActionEmployeeDeactivate        = "employee.deactivate"
	ActionEmployeeActivate          = "employee.activate"
	ActionEmployeeRoleChange        = "employee.role_change"
	ActionEmployeeContactCreate     = "employee.emergency_contact.create"
	ActionEmployeeContactUpdate     = "employee.emergency_contact.update"
	ActionEmployeeContactDelete     = "employee.emergency_contact.delete"
	ActionEmployeeProfileSelfUpdate = "employee.profile_self_update"

	ActionProjectCreate       = "project.create"
	ActionProjectUpdate       = "project.update"
	ActionProjectStatusChange = "project.status_change"

	ActionProjectAllocationCreate = "project_allocation.create"
	ActionProjectAllocationUpdate = "project_allocation.update"
	ActionProjectAllocationDelete = "project_allocation.delete"

	ActionLeaveTypeCreate = "leave_type.create"
	ActionLeaveTypeUpdate = "leave_type.update"

	ActionLeaveRequestApply   = "leave_request.apply"
	ActionLeaveRequestCancel  = "leave_request.cancel"
	ActionLeaveRequestApprove = "leave_request.approve"
	ActionLeaveRequestReject  = "leave_request.reject"
)
