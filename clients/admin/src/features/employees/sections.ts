export type EmployeeSectionGroup = 'Profile' | 'Work' | 'Access'

/** Sub-nav group order. */
export const EMPLOYEE_SECTION_GROUPS: EmployeeSectionGroup[] = ['Profile', 'Work', 'Access']

export interface EmployeeSection {
  /** URL segment under /employees/:id. */
  path: string
  label: string
  /** Heading the section is listed under in the sub-nav. */
  group: EmployeeSectionGroup
  /** Only shown/routable when the viewer may change this employee's role. */
  roleOnly?: boolean
}

/** Sub-pages of the employee detail layout; the first entry is the default (`/employees/:id` redirects to it). */
export const EMPLOYEE_SECTIONS: EmployeeSection[] = [
  { path: 'overview', group: 'Profile', label: 'Overview' },
  { path: 'employment', group: 'Profile', label: 'Employment' },
  { path: 'personal', group: 'Profile', label: 'Personal details' },
  { path: 'address', group: 'Profile', label: 'Address' },
  { path: 'emergency-contacts', group: 'Profile', label: 'Emergency contacts' },
  { path: 'assignments', group: 'Profile', label: 'Assignment history' },
  { path: 'allocations', group: 'Work', label: 'Allocations' },
  { path: 'status-logs', group: 'Work', label: 'Status logs' },
  { path: 'leave', group: 'Work', label: 'Leave' },
  { path: 'role', group: 'Access', label: 'Role', roleOnly: true },
]
