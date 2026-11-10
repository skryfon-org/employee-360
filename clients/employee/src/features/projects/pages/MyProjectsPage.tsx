import { getErrorMessage } from '@employee360/api-client'
import { EmptyState, Card, InlineAlert, PageContainer, PageHeader } from '@employee360/ui'
import { useSearchParams } from 'react-router-dom'
import { ProjectsCalendar } from '../components/ProjectsCalendar'
import { ProjectsPagination } from '../components/ProjectsPagination'
import { ProjectsTable } from '../components/ProjectsTable'
import { PAGE_SIZE, filterAllocations } from '../components/filterProjects'
import { TotalAllocationChip } from '../components/TotalAllocationChip'
import { ProjectsViewToggle, type ProjectsViewMode } from '../components/ProjectsViewToggle'
import { useMyProjectsQuery, type ProjectScope } from '../queries/projectQueries'

type ListTab = 'active' | 'upcoming' | 'past'

const TABS: { value: ListTab; label: string }[] = [
  { value: 'active', label: 'Active' },
  { value: 'upcoming', label: 'Upcoming' },
  { value: 'past', label: 'Past' },
]

const isTab = (v: string | null): v is ListTab => TABS.some((t) => t.value === v)

const EMPTY: Record<ListTab, { title: string; body: string }> = {
  active: {
    title: 'No active projects',
    body: 'You are not allocated to any project right now.',
  },
  upcoming: {
    title: 'No upcoming projects',
    body: 'You have no allocations starting in the future.',
  },
  past: {
    title: 'No past projects',
    body: 'You have no completed allocations yet.',
  },
}

interface ResultsProps {
  scope: ProjectScope
  view: 'list' | 'calendar'
  query?: string
  page?: number
  onPageChange?: (page: number) => void
  onClearSearch?: () => void
}

function Results({ scope, view, query = '', page = 1, onPageChange, onClearSearch }: ResultsProps) {
  const { data, isPending, isError, error, refetch } = useMyProjectsQuery(scope)
  const rows = data?.data?.allocations ?? []
  if (isPending)
    return (
      <p role="status" className="p-4 text-sm text-slate-600">
        Loading your projects…
      </p>
    )
  if (isError)
    return (
      <div className="flex flex-col items-start gap-3 p-4">
        <InlineAlert tone="error">{getErrorMessage(error, 'Could not load your projects.')}</InlineAlert>
        <button
          type="button"
          onClick={() => void refetch()}
          className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
        >
          Retry
        </button>
      </div>
    )
  if (view === 'calendar') return <ProjectsCalendar allocations={rows} />
  if (rows.length === 0) {
    const e = EMPTY[scope as ListTab]
    return (
      <EmptyState title={e.title} description={e.body} />
    )
  }
  const filtered = filterAllocations(rows, query)
  if (filtered.length === 0)
    return (
      <EmptyState
        title="No projects match your search"
        action={
          <button
            type="button"
            onClick={onClearSearch}
            className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
          >
            Clear search
          </button>
        }
      />
    )
  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const current = Math.min(Math.max(1, page), totalPages)
  const visible = filtered.slice((current - 1) * PAGE_SIZE, current * PAGE_SIZE)
  return (
    <>
      <ProjectsTable rows={visible} />
      <ProjectsPagination page={current} pageSize={PAGE_SIZE} totalItems={filtered.length} onPageChange={(p) => onPageChange?.(p)} />
    </>
  )
}

export default function MyProjectsPage() {
  const [params, setParams] = useSearchParams()
  const view: ProjectsViewMode = params.get('view') === 'calendar' ? 'calendar' : 'list'
  const rawTab = params.get('tab')
  const tab: ListTab = isTab(rawTab) ? rawTab : 'active'

  const q = params.get('q') ?? ''
  const rawPage = Number(params.get('page'))
  const page = Number.isInteger(rawPage) && rawPage >= 1 ? rawPage : 1

  const update = (changes: Record<string, string | null>) =>
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        for (const [k, v] of Object.entries(changes)) {
          if (v === null || v === '') next.delete(k)
          else next.set(k, v)
        }
        return next
      },
      { replace: true },
    )

  return (
    <PageContainer>
      <PageHeader
        title="My projects"
        description="Your project allocations."
        actions={<ProjectsViewToggle view={view} onChange={(v) => update({ view: v })} />}
      />
      <TotalAllocationChip scope={view === 'list' ? tab : 'all'} />
      {view === 'list' ? (
        <>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <label className="flex flex-1 flex-col gap-1 text-xs font-medium text-slate-600">
              <span className="sr-only">Search projects</span>
              <input
                type="search"
                aria-label="Search projects"
                placeholder="Search by project, code or role"
                value={q}
                onChange={(e) => update({ q: e.target.value, page: null })}
                className="h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-sm text-slate-900 placeholder:text-slate-500 focus:border-slate-900 focus:outline-none md:h-9"
              />
            </label>
            <select
              aria-label="Allocation status"
              value={tab}
              onChange={(e) => update({ tab: e.target.value, page: null })}
              className="h-11 rounded-sm border border-slate-300 bg-white px-3 text-sm text-slate-900 focus:border-slate-900 focus:outline-none md:h-9 sm:w-48"
            >
              {TABS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </div>
          <Card padded={false} role="region" aria-label="My projects" className="overflow-hidden">
            <Results
              scope={tab}
              view="list"
              query={q}
              page={page}
              onPageChange={(p) => update({ page: p === 1 ? null : String(p) })}
              onClearSearch={() => update({ q: null, page: null })}
            />
          </Card>
        </>
      ) : (
        <Card padded={false} role="region" aria-label="My projects calendar" className="overflow-hidden">
          <Results scope="all" view="calendar" />
        </Card>
      )}
    </PageContainer>
  )
}
