import type { AdminDashboard, TenantInfo } from '@employee360/api-client'
import { DepartmentsIcon, EmployeesIcon, MailIcon, PositionsIcon } from '../../../components/layout/NavIcons'
import { UserCheckIcon, UserOffIcon } from './DashboardIcons'
import { StatCards } from '@employee360/ui'
import { OrganisationCard } from './OrganisationCard'

/**
 * Shared by the admin and super-admin dashboards. `tenant` is only known to the super-admin
 * endpoint, so this is the single place that decides whether the organisation panel renders.
 */
export function DashboardBody({ data, tenant }: { data: AdminDashboard; tenant?: TenantInfo }) {
  return (
    <div className="flex flex-col gap-6">
      {tenant && <OrganisationCard tenant={tenant} />}
      <StatCards
        label="User statistics"
        stats={[
          { label: 'Total users', value: data.users?.total, description: 'All accounts in the organisation', icon: EmployeesIcon, to: '/employees' },
          { label: 'Active users', value: data.users?.active, description: 'Can sign in today', icon: UserCheckIcon },
          { label: 'Inactive users', value: data.users?.inactive, description: 'Access disabled', icon: UserOffIcon },
          { label: 'Pending invited', value: data.users?.pending_invited, description: 'Awaiting acceptance', icon: MailIcon, to: '/invitations' },
        ]}
      />
      <StatCards
        label="Organisation structure"
        columns={2}
        stats={[
          { label: 'Departments', value: data.departments, description: 'Teams across the organisation', icon: DepartmentsIcon, to: '/departments' },
          { label: 'Positions', value: data.positions, description: 'Defined roles and titles', icon: PositionsIcon, to: '/positions' },
        ]}
      />
    </div>
  )
}
