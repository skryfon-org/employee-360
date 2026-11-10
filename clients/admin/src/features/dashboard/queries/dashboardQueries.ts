import { useQuery } from '@tanstack/react-query'
import { fetchAdminDashboard, fetchSuperAdminDashboard } from '@employee360/api-client'

export const DASHBOARD_KEY = ['dashboard'] as const

/** Each hook only fetches when `enabled`, so a role never calls the other role's endpoint. */
export function useAdminDashboardQuery(enabled = true) {
  return useQuery({
    queryKey: [...DASHBOARD_KEY, 'admin'],
    queryFn: ({ signal }) => fetchAdminDashboard(signal),
    enabled,
  })
}

export function useSuperAdminDashboardQuery(enabled = true) {
  return useQuery({
    queryKey: [...DASHBOARD_KEY, 'super-admin'],
    queryFn: ({ signal }) => fetchSuperAdminDashboard(signal),
    enabled,
  })
}
