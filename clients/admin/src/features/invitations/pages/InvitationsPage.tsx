import { Link } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { useInvitationsQuery } from '../queries/invitationQueries'
import { useInvitationListParams } from '../hooks/useInvitationListParams'
import { InvitationsTableSkeleton } from '../components/InvitationsTableSkeleton'
import { InvitationsTable } from '../components/InvitationsTable'
import { InvitationsToolbar } from '../components/InvitationsToolbar'
import { InvitationsPagination } from '../components/InvitationsPagination'
import { INVITATION_STATUSES } from '../schemas/invitationListSchema'
import { Card, InlineAlert, PageHeader, PageContainer } from '@employee360/ui'

export default function InvitationsPage() {
  const { params, update } = useInvitationListParams()
  const { data, isPending, isError, error, isPlaceholderData } = useInvitationsQuery(params)
  const meta = data?.meta
  const totalItems = meta?.total_items ?? data?.data.length ?? 0
  const totalPages = Math.max(meta?.total_pages ?? 1, 1)
  const filtered = Boolean(params.status || params.search)

  return (
    <PageContainer>
      <PageHeader
        title="Invitations"
        description="Invite people to your organization and track their status."
        actions={
          <Link
            to="/invitations/new"
            className="inline-flex h-11 md:h-9 items-center rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
          >
            Create invitation
          </Link>
        }
      />
      <Card padded={false} aria-label="Invitations list" role="region" className="overflow-hidden">
        <InvitationsToolbar
          status={params.status}
          search={params.search}
          onStatusChange={(s) =>
            update({ status: s && (INVITATION_STATUSES as readonly string[]).includes(s) ? (s as typeof params.status) : undefined })
          }
          onSearchChange={(search) => update({ search })}
        />
        {isPending && <InvitationsTableSkeleton />}
        {isError && (
          <div className="p-4">
            <InlineAlert tone="error">{getErrorMessage(error, 'Could not load invitations.')}</InlineAlert>
          </div>
        )}
        {data && (
          <div className={isPlaceholderData ? 'opacity-60 transition-opacity' : undefined} aria-busy={isPlaceholderData}>
            <InvitationsTable invitations={data.data} filtered={filtered} />
          </div>
        )}
        {data && data.data.length > 0 && (
          <InvitationsPagination
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
