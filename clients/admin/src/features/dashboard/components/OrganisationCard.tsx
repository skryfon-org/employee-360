import type { TenantInfo } from '@employee360/api-client'
import { BuildingIcon } from '../../../components/layout/NavIcons'
import { formatCreated } from './formatCreated'

function StatusBadge({ active }: { active: boolean }) {
  return (
    <span
      className={`inline-flex h-6 items-center gap-1.5 rounded-sm border px-2 text-xs font-medium ${
        active ? 'border-green-300 bg-green-50 text-green-700' : 'border-slate-300 bg-slate-100 text-slate-600'
      }`}
    >
      <span aria-hidden="true" className={`h-1.5 w-1.5 rounded-full ${active ? 'bg-green-700' : 'bg-slate-600'}`} />
      {active ? 'Active' : 'Inactive'}
    </span>
  )
}

/**
 * Organisation summary. Rendered by DashboardBody only when a tenant is known (super-admin
 * dashboard); the admin endpoint does not expose it.
 */
export function OrganisationCard({ tenant }: { tenant: TenantInfo }) {
  return (
    <section
      aria-labelledby="org-title"
      className="flex flex-col gap-4 rounded-sm border border-line bg-surface p-4 sm:flex-row sm:items-center sm:justify-between sm:p-6"
    >
      <div className="flex min-w-0 items-center gap-4">
        <div
          aria-hidden="true"
          className="flex h-12 w-12 shrink-0 items-center justify-center rounded-sm border border-slate-200 bg-slate-100 text-slate-700"
        >
          <BuildingIcon width={24} height={24} />
        </div>
        <div className="min-w-0">
          <p id="org-title" className="text-xs font-medium uppercase tracking-wide text-slate-600">Organisation</p>
          <div className="mt-1 flex flex-wrap items-center gap-2">
            <h2 className="break-words text-xl font-semibold text-slate-900">{tenant.name}</h2>
            <StatusBadge active={Boolean(tenant.is_active)} />
          </div>
        </div>
      </div>
      <dl className="flex flex-col gap-1 border-t border-slate-200 pt-4 sm:border-l sm:border-t-0 sm:pl-6 sm:pt-0 sm:text-right">
        <dt className="text-xs font-medium text-slate-600">Created</dt>
        <dd className="text-sm font-semibold text-slate-900">{formatCreated(tenant.created_at)}</dd>
      </dl>
    </section>
  )
}
