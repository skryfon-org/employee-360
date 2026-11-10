import { useQuery } from '@tanstack/react-query'
import { fetchEmployeeDashboard } from '@employee360/api-client'

export const EMPLOYEE_DASHBOARD_KEY = ['dashboard', 'employee'] as const

export function useEmployeeDashboardQuery() {
  return useQuery({
    queryKey: EMPLOYEE_DASHBOARD_KEY,
    queryFn: ({ signal }) => fetchEmployeeDashboard(signal),
  })
}
