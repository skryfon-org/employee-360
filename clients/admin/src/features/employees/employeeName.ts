import type { EmployeeSummary } from './queries/employeeQueries'

export const fullName = (e: Pick<EmployeeSummary, 'first_name' | 'last_name' | 'email'>) =>
  [e.first_name, e.last_name].filter(Boolean).join(' ') || e.email || 'Employee'
