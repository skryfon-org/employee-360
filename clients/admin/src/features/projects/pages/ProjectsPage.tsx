import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { Card, FilterBar, InlineAlert, PageContainer, PageHeader } from '@employee360/ui'
import { useProjectsQuery } from '../queries/projectQueries'
import { useProjectListParams } from '../hooks/useProjectListParams'
import { useDebouncedValue } from '../../../hooks/useDebouncedValue'
import { PROJECT_STATUSES, PROJECT_STATUS_LABELS, type ProjectStatus } from '../schemas/projectSchemas'
import { ProjectsTable } from '../components/ProjectsTable'
import { DepartmentsPagination as Pagination } from '../../departments/components/DepartmentsPagination'

const INPUT =
  'h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 placeholder:text-slate-500 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9'

export default function ProjectsPage() {
  const { params, update } = useProjectListParams()
  const { data, isPending, isError, error, isPlaceholderData, refetch } = useProjectsQuery(params)
  const [search, setSearch] = useState(params.search)
  const debounced = useDebouncedValue(search, 300)

  useEffect(() => {
    if (debounced.trim() !== params.search) update({ search: debounced.trim() })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debounced])

  const rows = data?.data ?? []
  const totalItems = data?.meta?.total_items ?? rows.length
  const totalPages = Math.max(data?.meta?.total_pages ?? 1, 1)

  return (
    <PageContainer>
      <PageHeader
        title="Projects"
        description="Manage projects and allocate employees to them."
        actions={
          <Link
            to="/projects/new"
            className="inline-flex h-11 items-center rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
          >
            New project
          </Link>
        }
      />
      <Card padded={false} aria-label="Projects list" role="region" className="overflow-hidden">
        <FilterBar className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <div className="flex flex-col gap-1">
            <label htmlFor="project-search" className="text-xs font-medium text-slate-900">Search</label>
            <input
              id="project-search"
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Name or code"
              className={INPUT}
            />
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="project-status" className="text-xs font-medium text-slate-900">Status</label>
            <select
              id="project-status"
              value={params.status ?? ''}
              onChange={(e) => update({ status: (e.target.value || undefined) as ProjectStatus | undefined })}
              className={INPUT}
            >
              <option value="">All</option>
              {PROJECT_STATUSES.map((s) => (
                <option key={s} value={s}>{PROJECT_STATUS_LABELS[s]}</option>
              ))}
            </select>
          </div>
        </FilterBar>
        {isPending && <p role="status" className="p-4 text-sm text-slate-600">Loading projects…</p>}
        {isError && (
          <div className="flex flex-col items-start gap-3 p-4">
            <InlineAlert tone="error">{getErrorMessage(error, 'Could not load projects.')}</InlineAlert>
            <button
              type="button"
              onClick={() => void refetch()}
              className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
            >
              Retry
            </button>
          </div>
        )}
        {data && (
          <div className={isPlaceholderData ? 'opacity-60 transition-opacity' : undefined} aria-busy={isPlaceholderData}>
            <ProjectsTable projects={rows} filtered={Boolean(params.search || params.status)} />
          </div>
        )}
        {data && rows.length > 0 && (
          <Pagination
            page={params.page}
            pageSize={params.page_size}
            totalItems={totalItems}
            totalPages={totalPages}
            onPageChange={(page) => update({ page })}
            onPageSizeChange={(page_size) => update({ page_size })}
          />
        )}
      </Card>
    </PageContainer>
  )
}
