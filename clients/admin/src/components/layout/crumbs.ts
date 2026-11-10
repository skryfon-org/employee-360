import type { CrumbConfig } from '@employee360/ui'
import { ORGANIZATION_BASE, ORGANIZATION_TABS } from '../../features/tenants/tabs'

/** Route pattern -> crumb label. Add new routes (including `:param` detail routes) here. */
export const CRUMBS: CrumbConfig = {
  '/': 'Dashboard',
  '/invitations': 'Invitations',
  '/invitations/new': 'Invite user',
  '/employees': 'Employees',
  '/employees/:id': 'Employee',
  '/employees/:id/overview': 'Overview',
  '/employees/:id/employment': 'Employment',
  '/employees/:id/personal': 'Personal details',
  '/employees/:id/address': 'Address',
  '/employees/:id/emergency-contacts': 'Emergency contacts',
  '/employees/:id/assignments': 'Assignment history',
  '/employees/:id/allocations': 'Allocations',
  '/employees/:id/status-logs': 'Status logs',
  '/employees/:id/leave': 'Leave',
  '/employees/:id/role': 'Role',
  '/departments': 'Departments',
  '/positions': 'Positions',
  '/projects': 'Projects',
  '/projects/new': 'New project',
  '/projects/:id': 'Project',
  '/projects/:id/edit': 'Edit',
  '/status-logs': 'Status logs',
  '/holidays': 'Holidays',
  '/holidays/categories': 'Categories',
  '/holidays/settings': 'Calendar settings',
  '/leave': 'Leave',
  '/leave/types': 'Leave types',
  '/audit-logs': 'Audit log',
  '/settings/organization': 'Organization',
  // One crumb per organization tab (Organization > Domains).
  ...Object.fromEntries(ORGANIZATION_TABS.map((t) => [`${ORGANIZATION_BASE}/${t.path}`, t.label])),
}
