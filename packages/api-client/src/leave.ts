import { apiRequest } from './client.ts';
import { unwrapListResponse, unwrapSingleEntity, type ApiEnvelope } from './unwrap.ts';
import type { PageMeta } from './auth.ts';

import type {
  GithubComSkryfonOrgEmployee360BackendInternalTypesLeaveApplyLeaveRequest as GenApply,
  GithubComSkryfonOrgEmployee360BackendInternalTypesLeaveLeaveApproverResponse as GenApprover,
  GithubComSkryfonOrgEmployee360BackendInternalTypesLeaveLeaveRequestResponse as GenRequest,
  GithubComSkryfonOrgEmployee360BackendInternalTypesLeaveLeaveRequestResponseDayPart as GenDayPart,
  GithubComSkryfonOrgEmployee360BackendInternalTypesLeaveLeaveRequestResponseStatus as GenStatus,
  GithubComSkryfonOrgEmployee360BackendInternalTypesLeaveLeaveTypeResponse as GenType,
} from './generated/models/index.ts';

// Entity types derive from the Orval-generated models (source of truth: backend/docs/swagger.json,
// regenerate with `pnpm generate:api`). The swagger marks every response field optional, but the
// server always sends them, so they are tightened with Required<> here; decided_* can be null.
// The request wrappers below stay hand-written because they unwrap the response envelope.

export type LeaveDayPart = GenDayPart;
export type LeaveStatus = GenStatus;

export type LeaveType = Required<Pick<GenType, 'id' | 'name' | 'code' | 'description' | 'is_active'>>;

export type LeaveApprover = Required<GenApprover>;

export type LeaveRequest = Required<Omit<GenRequest, 'decided_by' | 'decided_by_name' | 'decided_at'>> & {
  decided_by?: string | null;
  /** Decider's display name; the key is always present, null while undecided. */
  decided_by_name: string | null;
  decided_at?: string | null;
};

export type ApplyLeaveRequest = GenApply;

export interface ListLeaveRequestsParams {
  page?: number;
  page_size?: number;
  status?: LeaveStatus;
  /** Sent as `leave_type_id`; a malformed UUID is a 400, an unknown valid one an empty list. */
  leaveTypeId?: string;
  /** YYYY-MM-DD, inclusive. Matches requests whose end_date >= from. */
  from?: string;
  /** YYYY-MM-DD, inclusive. Matches requests whose start_date <= to. `to` before `from` is a 400. */
  to?: string;
  /** Team list only; sent as `exclude_status`. Ignored by the server if `status` is set. */
  excludeStatus?: LeaveStatus;
}

/** Admin listing filters (`GET /leave-requests`). */
export interface ListAllLeaveRequestsParams extends ListLeaveRequestsParams {
  /** Applicant user UUID; sent as `employee_id`. */
  employeeId?: string;
}

export interface LeaveTypeInput {
  name: string;
  code: string;
  description?: string;
  /** Defaults to true on the server when omitted. */
  is_active?: boolean;
}

export interface LeaveTypePatch {
  name?: string;
  code?: string;
  description?: string;
  is_active?: boolean;
}

/** Stable error codes returned by the leave endpoints. */
export const LEAVE_ERROR_CODES = {
  OVERLAP: 'leave_overlap',
  START_TOO_OLD: 'leave_start_too_old',
  INVALID_APPROVER: 'invalid_approver',
  TYPE_INACTIVE: 'leave_type_inactive',
  NOT_PENDING: 'leave_not_pending',
  NOT_CANCELLABLE: 'leave_not_cancellable',
} as const;

/** `GET /api/v1/leave-types` — active leave types. */
export async function listLeaveTypes(signal?: AbortSignal): Promise<LeaveType[]> {
  const res = await apiRequest<ApiEnvelope<LeaveType[]>>({ url: '/api/v1/leave-types', method: 'GET', signal });
  return unwrapListResponse(res).data;
}

/** `GET /api/v1/me/leave-approvers` — the caller's manager plus tenant admins. */
export async function listLeaveApprovers(signal?: AbortSignal): Promise<LeaveApprover[]> {
  const res = await apiRequest<ApiEnvelope<LeaveApprover[]>>({ url: '/api/v1/me/leave-approvers', method: 'GET', signal });
  return unwrapListResponse(res).data;
}

/** `GET /api/v1/organization/weekly-off-days` — tenant weekly off days (0=Sunday..6=Saturday). */
export async function getWeeklyOffDays(signal?: AbortSignal): Promise<number[]> {
  const res = await apiRequest<ApiEnvelope<{ weekly_off_days: number[] }>>({ url: '/api/v1/organization/weekly-off-days', method: 'GET', signal });
  return unwrapSingleEntity(res).weekly_off_days;
}

/** `POST /api/v1/me/leave-requests`. */
export async function applyLeave(body: ApplyLeaveRequest): Promise<LeaveRequest> {
  return unwrapSingleEntity(await apiRequest<ApiEnvelope<LeaveRequest>>({ url: '/api/v1/me/leave-requests', method: 'POST', data: body }));
}

function toQuery(params?: ListLeaveRequestsParams) {
  if (!params) return undefined;
  const { leaveTypeId, excludeStatus, ...rest } = params;
  return { ...rest, leave_type_id: leaveTypeId || undefined, exclude_status: excludeStatus || undefined };
}

/** `GET /api/v1/me/leave-requests`. */
export async function listMyLeaveRequests(params?: ListLeaveRequestsParams, signal?: AbortSignal) {
  const res = await apiRequest<ApiEnvelope<LeaveRequest[], PageMeta>>({ url: '/api/v1/me/leave-requests', method: 'GET', params: toQuery(params), signal });
  return unwrapListResponse<LeaveRequest, PageMeta>(res);
}

/** `POST /api/v1/me/leave-requests/:id/cancel`. */
export async function cancelLeaveRequest(id: string): Promise<LeaveRequest> {
  return unwrapSingleEntity(await apiRequest<ApiEnvelope<LeaveRequest>>({ url: `/api/v1/me/leave-requests/${id}/cancel`, method: 'POST' }));
}

/** `GET /api/v1/leave-requests/team` — requests addressed to the caller as approver. */
export async function listTeamLeaveRequests(params?: ListLeaveRequestsParams, signal?: AbortSignal) {
  const res = await apiRequest<ApiEnvelope<LeaveRequest[], PageMeta>>({ url: '/api/v1/leave-requests/team', method: 'GET', params: toQuery(params), signal });
  return unwrapListResponse<LeaveRequest, PageMeta>(res);
}

/** `POST /api/v1/leave-requests/:id/approve`. */
export async function approveLeaveRequest(id: string, comment?: string): Promise<LeaveRequest> {
  return unwrapSingleEntity(
    await apiRequest<ApiEnvelope<LeaveRequest>>({ url: `/api/v1/leave-requests/${id}/approve`, method: 'POST', data: { comment: comment ?? '' } }),
  );
}

/** `POST /api/v1/leave-requests/:id/reject`. */
export async function rejectLeaveRequest(id: string, comment?: string): Promise<LeaveRequest> {
  return unwrapSingleEntity(
    await apiRequest<ApiEnvelope<LeaveRequest>>({ url: `/api/v1/leave-requests/${id}/reject`, method: 'POST', data: { comment: comment ?? '' } }),
  );
}

/** `GET /api/v1/leave-types?include_inactive=true` — every leave type (admin only; ignored for others). */
export async function listAllLeaveTypes(signal?: AbortSignal): Promise<LeaveType[]> {
  const res = await apiRequest<ApiEnvelope<LeaveType[]>>({ url: '/api/v1/leave-types', method: 'GET', params: { include_inactive: true }, signal });
  return unwrapListResponse(res).data;
}

/** `POST /api/v1/leave-types` (admin). */
export async function createLeaveType(body: LeaveTypeInput): Promise<LeaveType> {
  return unwrapSingleEntity(await apiRequest<ApiEnvelope<LeaveType>>({ url: '/api/v1/leave-types', method: 'POST', data: body }));
}

/** `PATCH /api/v1/leave-types/:id` (admin). Absent fields are unchanged; deactivate with `is_active: false`. */
export async function updateLeaveType(id: string, body: LeaveTypePatch): Promise<LeaveType> {
  return unwrapSingleEntity(await apiRequest<ApiEnvelope<LeaveType>>({ url: `/api/v1/leave-types/${id}`, method: 'PATCH', data: body }));
}

/** `GET /api/v1/leave-requests` — every request in the tenant (admin only). */
export async function listAllLeaveRequests(params?: ListAllLeaveRequestsParams, signal?: AbortSignal) {
  const { employeeId, ...rest } = params ?? {};
  const res = await apiRequest<ApiEnvelope<LeaveRequest[], PageMeta>>({
    url: '/api/v1/leave-requests',
    method: 'GET',
    params: { ...toQuery(rest), employee_id: employeeId || undefined },
    signal,
  });
  return unwrapListResponse<LeaveRequest, PageMeta>(res);
}
