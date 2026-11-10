import type { ComponentType, SVGProps } from 'react'
import { AuditLogIcon, EmployeesIcon, BuildingIcon, DashboardIcon, DepartmentsIcon, HolidayIcon, LeaveIcon, MailIcon, PositionsIcon, ProjectsIcon, StatusLogIcon } from './NavIcons'

export interface NavItem {
  to: string
  label: string
  /** Icon shown beside the label, and alone when the sidebar is collapsed. */
  icon: ComponentType<SVGProps<SVGSVGElement>>
  /** When set, the item is shown only to users holding one of these roles. */
  roles?: string[]
}

/** Add new admin sections here; the sidebar renders this list. */
export const NAV_ITEMS: NavItem[] = [
  { to: '/', label: 'Dashboard', icon: DashboardIcon },
  { to: '/invitations', label: 'Invitations', icon: MailIcon },
  { to: '/employees', label: 'Employees', icon: EmployeesIcon, roles: ['admin', 'super_admin'] },
  { to: '/departments', label: 'Departments', icon: DepartmentsIcon, roles: ['admin', 'super_admin'] },
  { to: '/positions', label: 'Positions', icon: PositionsIcon, roles: ['admin', 'super_admin'] },
  { to: '/projects', label: 'Projects', icon: ProjectsIcon, roles: ['admin', 'super_admin'] },
  { to: '/status-logs', label: 'Status logs', icon: StatusLogIcon, roles: ['admin', 'super_admin'] },
  { to: '/holidays', label: 'Holidays', icon: HolidayIcon, roles: ['admin', 'super_admin'] },
  { to: '/leave', label: 'Leave', icon: LeaveIcon, roles: ['admin', 'super_admin'] },
  { to: '/audit-logs', label: 'Audit log', icon: AuditLogIcon, roles: ['admin', 'super_admin'] },
  { to: '/settings/organization', label: 'Organization', icon: BuildingIcon, roles: ['admin', 'super_admin'] },
]

export function visibleNavItems(roles: string[] | undefined): NavItem[] {
  return NAV_ITEMS.filter((i) => !i.roles || i.roles.some((r) => roles?.includes(r)))
}
