import { NavLink, Outlet } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { InlineAlert, PageContainer, PageHeader, Skeleton, SkeletonRegion } from '@employee360/ui'
import { useMyProfileQuery } from '../queries/profileQueries'
import { MissingFieldsAlert } from '../components/MissingFieldsAlert'
import { secondaryButtonCls } from '../components/fields'
import { PROFILE_SECTIONS } from '../sections'

const FOCUS = 'focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'

/** Layout for /profile/*: header + NavLink sub-nav; the profile is fetched once and shared via Outlet context. */
export default function ProfileLayout() {
  const { data, isPending, isError, error, refetch } = useMyProfileQuery()

  if (isPending) {
    return (
      <PageContainer>
        <SkeletonRegion label="Loading your profile…" className="flex flex-col gap-4">
          <Skeleton className="h-7 w-64" />
          <Skeleton className="h-64 w-full" />
        </SkeletonRegion>
      </PageContainer>
    )
  }
  if (isError || !data) {
    return (
      <PageContainer>
        <InlineAlert tone="error">{getErrorMessage(error, 'Could not load your profile.')}</InlineAlert>
        <div>
          <button type="button" className={secondaryButtonCls} onClick={() => void refetch()}>
            Retry
          </button>
        </div>
      </PageContainer>
    )
  }

  return (
    <PageContainer>
      <PageHeader title="My profile" description="Keep your personal details and emergency contacts up to date." />
      <MissingFieldsAlert fields={data.missing_fields} />
      <nav aria-label="Profile sections" className="-mx-1 overflow-x-auto border-b border-slate-200 px-1">
        <ul className="flex min-w-max gap-1">
          {PROFILE_SECTIONS.map((s) => (
            <li key={s.path}>
              <NavLink
                to={s.path}
                className={({ isActive }) =>
                  `-mb-px inline-flex h-11 items-center border-b-2 px-3 text-sm md:h-10 ${FOCUS} ${
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
      </nav>
      {/* Only the section body scrolls; header and sub-nav stay in view. Edit/Save footers are sticky inside it. */}
      <div className="max-h-[max(24rem,calc(100vh-18rem))] min-w-0 overflow-x-auto overflow-y-auto">
        <Outlet context={data} />
      </div>
    </PageContainer>
  )
}
