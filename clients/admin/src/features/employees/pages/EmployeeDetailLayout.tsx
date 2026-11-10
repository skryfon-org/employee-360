import { useState } from 'react'
import { Link, NavLink, Outlet, useParams } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { InlineAlert, PageContainer, PageHeader, Skeleton, SkeletonRegion } from '@employee360/ui'
import { useAuthStore } from '../../../stores/authStore'
import { useEmployeeUiStore } from '../../../stores/employeeUiStore'
import { useEmployeeQuery } from '../queries/employeeQueries'
import { EmployeeStatusBadge } from '../components/EmployeeStatusBadge'
import { ToggleActiveDialog } from '../components/ToggleActiveDialog'
import { dangerButtonCls, secondaryButtonCls } from '../components/fields'
import { fullName } from '../employeeName'
import { EMPLOYEE_SECTION_GROUPS, EMPLOYEE_SECTIONS } from '../sections'
import type { EmployeeDetailContext } from '../hooks/useEmployeeDetail'

const FOCUS = 'focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'

/** Layout for /employees/:id/*: header (status, activate/deactivate, role), NavLink sub-nav, and the section Outlet. */
export default function EmployeeDetailLayout() {
  const { id } = useParams()
  const { data, isPending, isError, error, refetch } = useEmployeeQuery(id)
  const viewer = useAuthStore((s) => s.user)
  const listSearch = useEmployeeUiStore((s) => s.listSearch)
  const [toggling, setToggling] = useState(false)

  const back = (
    <Link to={`/employees${listSearch ? `?${listSearch}` : ''}`} className={`${secondaryButtonCls} inline-flex items-center`}>
      Back to employees
    </Link>
  )

  if (isPending) {
    return (
      <PageContainer>
        <SkeletonRegion label="Loading employee…" className="flex flex-col gap-4">
          <Skeleton className="h-7 w-64" />
          <Skeleton className="h-64 w-full" />
        </SkeletonRegion>
      </PageContainer>
    )
  }
  if (isError || !data) {
    return (
      <PageContainer>
        <InlineAlert tone="error">{getErrorMessage(error, 'Could not load employee.')}</InlineAlert>
        <div className="flex gap-2">
          <button type="button" className={secondaryButtonCls} onClick={() => void refetch()}>
            Retry
          </button>
          {back}
        </div>
      </PageContainer>
    )
  }

  const viewerIsSuper = viewer?.roles.includes('super_admin') ?? false
  const targetIsSuper = data.roles?.includes('super_admin') ?? false
  const readOnly = targetIsSuper && !viewerIsSuper
  const canChangeRole = viewerIsSuper && !targetIsSuper && viewer?.id !== data.id
  const name = fullName(data)
  const formKey = `${data.id}:${data.created_at}:${JSON.stringify(data.profile ?? {})}:${data.department_id}:${data.position_id}:${data.first_name}:${data.last_name}`
  const context: EmployeeDetailContext = { employee: data, readOnly, canChangeRole, formKey }

  return (
    <PageContainer>
      <PageHeader
        title={name}
        description={data.email}
        actions={
          <>
            {back}
            {!readOnly && data.id !== viewer?.id && (
              <button type="button" className={data.is_active ? dangerButtonCls : secondaryButtonCls} onClick={() => setToggling(true)}>
                {data.is_active ? 'Deactivate' : 'Activate'}
              </button>
            )}
          </>
        }
      />
      <div className="flex flex-wrap items-center gap-2">
        <EmployeeStatusBadge active={data.is_active} />
        {(data.roles ?? []).map((r) => (
          <span key={r} className="inline-flex h-6 items-center rounded-sm border border-slate-300 bg-slate-100 px-2 text-xs font-medium text-slate-900">
            {r === 'super_admin' ? 'Super admin' : r.charAt(0).toUpperCase() + r.slice(1)}
          </span>
        ))}
        {!data.profile_completed && <span className="text-xs text-slate-600">Profile incomplete</span>}
      </div>
      {readOnly && <InlineAlert tone="neutral">Only a super admin can edit a super admin account.</InlineAlert>}
      {/* Tab row split into labelled groups (Profile / Work / Access); scrolls sideways on small screens. */}
      <nav aria-label="Employee sections" className="-mx-1 overflow-x-auto border-b border-slate-200 px-1">
        <ul className="flex min-w-max items-end gap-4">
          {EMPLOYEE_SECTION_GROUPS.map((group) => {
            const items = EMPLOYEE_SECTIONS.filter((s) => s.group === group && (!s.roleOnly || canChangeRole))
            if (items.length === 0) return null
            return (
              <li key={group} className="flex flex-col border-l border-slate-200 pl-4 first:border-l-0 first:pl-0">
                <span aria-hidden="true" className="px-3 pb-0.5 text-[0.6875rem] font-semibold uppercase tracking-wide text-slate-600">{group}</span>
                <ul aria-label={group} className="flex gap-1">
                  {items.map((s) => (
                    <li key={s.path}>
                      <NavLink
                        to={s.path}
                        className={({ isActive }) =>
                          `-mb-px inline-flex h-11 items-center whitespace-nowrap border-b-2 px-3 text-sm md:h-10 ${FOCUS} ${
                            isActive
                              ? 'border-slate-900 font-semibold text-slate-900'
                              : 'border-transparent text-slate-700 hover:bg-slate-100 hover:text-slate-900'
                          }`
                        }
                      >
                        {s.label}
                      </NavLink>
                    </li>
                  ))}
                </ul>
              </li>
            )
          })}
        </ul>
      </nav>
      {/* Only the section body scrolls; header and sub-nav stay in view. Save/Edit footers are sticky inside it. */}
      <div className="max-h-[max(24rem,calc(100vh-14rem))] min-w-0 overflow-y-auto overflow-x-auto">
        <Outlet context={context} />
      </div>
      {toggling && (
        <ToggleActiveDialog employeeId={data.id as string} name={name} activate={!data.is_active} onClose={() => setToggling(false)} />
      )}
    </PageContainer>
  )
}
