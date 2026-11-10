import { Link, useParams } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { Card, InlineAlert, PageContainer, PageHeader } from '@employee360/ui'
import { useProjectAllocationsQuery, useProjectQuery } from '../queries/projectQueries'
import { ProjectStatusBadge } from '../components/ProjectStatusBadge'
import { ProjectAllocations } from '../components/ProjectAllocations'
import { DefinitionList, secondaryButtonCls } from '../../employees/components/fields'

export default function ProjectDetailPage() {
  const { id } = useParams()
  const { data, isPending, isError, error, refetch } = useProjectQuery(id)
  const allocations = useProjectAllocationsQuery(id, 'active')

  const back = (
    <Link to="/projects" className={`${secondaryButtonCls} inline-flex items-center`}>
      Back to projects
    </Link>
  )
  if (isPending) {
    return (
      <PageContainer>
        <p role="status" className="text-sm text-slate-600">Loading project…</p>
      </PageContainer>
    )
  }
  if (isError || !data) {
    return (
      <PageContainer>
        <InlineAlert tone="error">{getErrorMessage(error, 'Could not load project.')}</InlineAlert>
        <div className="flex gap-2">
          <button type="button" className={secondaryButtonCls} onClick={() => void refetch()}>
            Retry
          </button>
          {back}
        </div>
      </PageContainer>
    )
  }

  // The server returns only allocations covering today (`state=active`); future and ended rows are excluded.
  // "Allocated employees" is the number of distinct people among those active allocations.
  const active = allocations.data ?? []
  const headcount = new Set(active.map((a) => a.user_id)).size
  const count = (n: number) => (allocations.isSuccess ? String(n) : '-')

  return (
    <PageContainer>
      <PageHeader
        title={data.name ?? 'Project'}
        description={data.client_name ? `Client: ${data.client_name}` : 'Project details and allocations.'}
        actions={
          <>
            {back}
            <Link to={`/projects/${data.id}/edit`} className={`${secondaryButtonCls} inline-flex items-center`}>
              Edit project
            </Link>
          </>
        }
      />
      <div className="flex flex-wrap items-center gap-2">
        <span className="inline-flex h-6 items-center rounded-sm border border-slate-300 bg-slate-100 px-2 text-xs font-medium text-slate-900">
          {data.code}
        </span>
        <ProjectStatusBadge status={data.status} />
      </div>
      <aside aria-label="Project summary" className="flex flex-col gap-4 rounded-sm border border-line bg-surface p-4">
        <h2 className="text-base font-semibold text-ink">Summary</h2>
        <DefinitionList
          className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4"
          rows={[
            ['Code', data.code],
            ['Status', <ProjectStatusBadge key="s" status={data.status} />],
            ['Start date', data.start_date],
            ['End date', data.end_date ?? 'Ongoing'],
            ['Client', data.client_name],
            ['Active allocations', count(active.length)],
            ['Allocated employees', count(headcount)],
          ]}
        />
        <div className="border-t border-line pt-4">
          <p className="text-xs font-medium text-ink-muted">Description</p>
          <p className="mt-1 whitespace-pre-line break-words text-sm text-ink">{data.description || '-'}</p>
        </div>
      </aside>
      <Card>
        <ProjectAllocations
          projectId={data.id as string} archived={data.status === 'archived'}
          projectStartDate={data.start_date}
          projectEndDate={data.end_date}
        />
      </Card>
    </PageContainer>
  )
}
