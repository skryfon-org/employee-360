export { apiClient, apiRequest, configureApiClient, configureAuth } from './client.ts';
export type { ApiClientConfig, AuthConfig } from './client.ts';

export {
  getSession,
  setSession,
  clearSession,
  subscribeSession,
  setSessionStorage,
} from './session.ts';
export type { Session, SessionStorageAdapter } from './session.ts';
export {
  createLocalStorageSessionAdapter,
  DEFAULT_SESSION_STORAGE_KEY,
} from './sessionStorage.ts';

export { unwrapSingleEntity, unwrapListResponse } from './unwrap.ts';
export type { ApiEnvelope } from './unwrap.ts';
export { getErrorMessage, getErrorCode, isConflictError, INVITATION_ERROR_CODES, EMAIL_DOMAIN_NOT_ALLOWED, TENANT_ERROR_CODES } from './errors.ts';
export { resolveInvitationRedirect, invitationAppForRole } from './invitationRouting.ts';
export type { InvitationApp, InvitationRedirect } from './invitationRouting.ts';

export {
  login,
  refresh,
  logout,
  forgotPassword,
  resetPassword,
  inviteUser,
  listInvitations,
  listRoles,
  resendInvitation,
  revokeInvitation,
  acceptInvitation,
  validateInvitation,
} from './auth.ts';
export type {
  LoginRequest,
  LoginResponse,
  ForgotPasswordRequest,
  ResetPasswordRequest,
  InviteUserRequest,
  AcceptInvitationRequest,
  ValidateInvitationResponse,
  Invitation,
  InvitationListItem,
  PageMeta,
  ListInvitationsParams,
  RoleOption,
} from './auth.ts';

export { fetchHealth, useHealth } from './health.ts';
export type { HealthResponse } from './health.ts';

export * from './generated/models/index.ts';
export * from './generated/hooks/index.ts';

export { fetchAdminDashboard, fetchSuperAdminDashboard, fetchEmployeeDashboard } from './dashboard.ts';
export type { EmployeeDashboard, EmployeeDashboardLeaveItem, AdminDashboard, SuperAdminDashboard, RoleUserCount, TenantInfo } from './dashboard.ts';

export {
  getTenant,
  renameTenant,
  listTenantDomains,
  addTenantDomain,
  updateTenantDomain,
  removeTenantDomain,
} from './tenants.ts';
export type { Tenant, TenantDetail, TenantDomain } from './tenants.ts';


export {
  listProjects,
  getProject,
  suggestProjectCode,
  createProject,
  updateProject,
  listProjectAllocations,
  createAllocation,
  updateAllocation,
  deleteAllocation,
  getEmployeeAllocations,
  getMyProjects,
} from './projects.ts';
export type {
  ProjectCodeSuggestion,
  Project,
  CreateProjectRequest,
  UpdateProjectRequest,
  ProjectAllocation,
  AllocationWriteResult,
  CreateAllocationRequest,
  UpdateAllocationRequest,
  UserAllocation,
  UserAllocations,
  ListProjectsParams,
  EmployeeAllocationScope,
} from './projects.ts';

export {
  listLeaveTypes,
  listLeaveApprovers,
  getWeeklyOffDays,
  applyLeave,
  listMyLeaveRequests,
  cancelLeaveRequest,
  listTeamLeaveRequests,
  approveLeaveRequest,
  rejectLeaveRequest,
  listAllLeaveTypes,
  createLeaveType,
  updateLeaveType,
  listAllLeaveRequests,
  LEAVE_ERROR_CODES,
} from './leave.ts';
export type {
  LeaveType,
  LeaveApprover,
  LeaveRequest,
  LeaveDayPart,
  LeaveStatus,
  ApplyLeaveRequest,
  ListLeaveRequestsParams,
  ListAllLeaveRequestsParams,
  LeaveTypeInput,
  LeaveTypePatch,
} from './leave.ts';
