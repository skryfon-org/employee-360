import type { ComponentType } from 'react'
import type { TenantDetail } from '@employee360/api-client'
import { useOutletContext } from 'react-router-dom'
import { RenameTenantForm } from './components/RenameTenantForm'
import { DomainsManager } from './components/DomainsManager'
import { EmployeeCodesPanel } from '../organization-settings/components/EmployeeCodesPanel'
import { WeeklyOffDaysPanel } from '../organization-settings/components/WeeklyOffDaysPanel'
import { WelcomeEmailPanel } from '../organization-settings/components/WelcomeEmailPanel'

export type OrganizationData = TenantDetail

export interface OrganizationTab {
  key: string
  label: string
  /** URL segment under /settings/organization. */
  path: string
  /** Roles that may see and open the tab. */
  roles: string[]
  /** True when the tab renders tenant detail (GET /tenant, super_admin-only on the backend). */
  requiresTenant: boolean
  component: ComponentType<{ tenant: OrganizationData | undefined }>
}

const GeneralTab = ({ tenant }: { tenant: OrganizationData | undefined }) => <RenameTenantForm name={tenant?.name ?? ''} />
const DomainsTab = ({ tenant }: { tenant: OrganizationData | undefined }) => <DomainsManager domains={tenant?.domains ?? []} />
const EmployeeCodesTab = () => <EmployeeCodesPanel />
const WorkingDaysTab = () => <WeeklyOffDaysPanel />
const WelcomeEmailTab = () => <WelcomeEmailPanel />

/** Add a future section (Branding, Departments, ...) with one line here; the first tab a user may see is their default. */
export const ORGANIZATION_TABS: OrganizationTab[] = [
  { key: 'general', label: 'General', path: 'general', roles: ['super_admin'], requiresTenant: true, component: GeneralTab },
  { key: 'domains', label: 'Domains', path: 'domains', roles: ['super_admin'], requiresTenant: true, component: DomainsTab },
  { key: 'employee-codes', label: 'Employee codes', path: 'employee-codes', roles: ['admin', 'super_admin'], requiresTenant: false, component: EmployeeCodesTab },
  { key: 'working-days', label: 'Working days', path: 'working-days', roles: ['admin', 'super_admin'], requiresTenant: false, component: WorkingDaysTab },
  { key: 'welcome-email', label: 'Welcome email', path: 'welcome-email', roles: ['admin', 'super_admin'], requiresTenant: false, component: WelcomeEmailTab },
]

export const ORGANIZATION_BASE = '/settings/organization'

/** Tabs the given roles may open, in registry order. */
export function visibleOrganizationTabs(roles: string[] | undefined): OrganizationTab[] {
  return ORGANIZATION_TABS.filter((t) => t.roles.some((r) => roles?.includes(r)))
}

/** Route element for a tab: reads the tenant loaded once by OrganizationPage. */
export function OrganizationTabRoute({ tab }: { tab: OrganizationTab }) {
  const tenant = useOutletContext<OrganizationData | undefined>()
  const Component = tab.component
  return <Component tenant={tenant} />
}
