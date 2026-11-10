import { unwrapSingleEntity } from './unwrap.ts';
import {
  getApiV1DashboardAdmin,
  getApiV1DashboardEmployee,
  getApiV1DashboardSuperAdmin,
} from './generated/hooks/dashboard/dashboard.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesDashboardAdminDashboardResponse as AdminDashboard } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesDashboardAdminDashboardResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesDashboardSuperAdminDashboardResponse as SuperAdminDashboard } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesDashboardSuperAdminDashboardResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesDashboardRoleUserCountResponse as RoleUserCount } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesDashboardRoleUserCountResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesDashboardTenantInfoResponse as TenantInfo } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesDashboardTenantInfoResponse.ts';

import type { GithubComSkryfonOrgEmployee360BackendInternalTypesDashboardEmployeeDashboardResponse as EmployeeDashboard } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesDashboardEmployeeDashboardResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesDashboardEmployeeLeaveItem as EmployeeDashboardLeaveItem } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesDashboardEmployeeLeaveItem.ts';

export type { EmployeeDashboard, EmployeeDashboardLeaveItem };
export type { AdminDashboard, SuperAdminDashboard, RoleUserCount, TenantInfo };

/** `GET /api/v1/dashboard/admin` — role `admin` only. */
export async function fetchAdminDashboard(signal?: AbortSignal): Promise<AdminDashboard> {
  return unwrapSingleEntity(await getApiV1DashboardAdmin(signal));
}

/** `GET /api/v1/dashboard/super-admin` — role `super_admin` only. */
export async function fetchSuperAdminDashboard(signal?: AbortSignal): Promise<SuperAdminDashboard> {
  return unwrapSingleEntity(await getApiV1DashboardSuperAdmin(signal));
}

/** `GET /api/v1/dashboard/employee` — the caller's own summary. */
export async function fetchEmployeeDashboard(signal?: AbortSignal): Promise<EmployeeDashboard> {
  return unwrapSingleEntity(await getApiV1DashboardEmployee(signal));
}
