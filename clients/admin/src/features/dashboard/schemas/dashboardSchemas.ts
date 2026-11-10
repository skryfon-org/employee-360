import { z } from 'zod'

export const dashboardRoles = z.enum(['admin', 'super_admin'])
export type DashboardRole = z.infer<typeof dashboardRoles>

/** super_admin wins when a user holds both roles. */
export function resolveDashboardRole(roles: string[] | undefined): DashboardRole | null {
  if (roles?.includes('super_admin')) return 'super_admin'
  if (roles?.includes('admin')) return 'admin'
  return null
}
