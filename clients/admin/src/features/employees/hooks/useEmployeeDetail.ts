import { useOutletContext } from 'react-router-dom'
import type { EmployeeDetail } from '../queries/employeeQueries'

export interface EmployeeDetailContext {
  employee: EmployeeDetail
  /** True when an admin views a super_admin account: all editing is disabled. */
  readOnly: boolean
  /** True only for a super admin viewing another, non-super-admin employee. */
  canChangeRole: boolean
  /** Changes whenever server data is replaced from elsewhere; use as `key` so forms remount with fresh defaults. */
  formKey: string
}

/** Employee loaded once by EmployeeDetailLayout, shared with every section page through Outlet context. */
export function useEmployeeDetail() {
  return useOutletContext<EmployeeDetailContext>()
}
