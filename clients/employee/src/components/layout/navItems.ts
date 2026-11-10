import type { ComponentType, SVGProps } from 'react'
import { HolidayIcon, StatusIcon, HomeIcon, ProfileIcon, ProjectsIcon, LeaveIcon } from './NavIcons'

export interface NavItem {
  to: string
  label: string
  /** Icon shown beside the label, and alone when the sidebar is collapsed. */
  icon: ComponentType<SVGProps<SVGSVGElement>>
}

/** Add new employee sections here; the sidebar renders this list. */
export const NAV_ITEMS: NavItem[] = [
  { to: '/', label: 'Home', icon: HomeIcon },
  { to: '/holidays', label: 'Holidays', icon: HolidayIcon },
  { to: '/projects', label: 'My projects', icon: ProjectsIcon },
  { to: '/status', label: 'Daily status', icon: StatusIcon },
  { to: '/leave', label: 'Leave', icon: LeaveIcon },
  { to: '/profile', label: 'My profile', icon: ProfileIcon },
]
