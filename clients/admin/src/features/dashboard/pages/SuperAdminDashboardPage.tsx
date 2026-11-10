import { PageHeader, PageContainer } from '@employee360/ui'
import { useSuperAdminDashboardQuery } from '../queries/dashboardQueries'
import { DashboardBody } from '../components/DashboardBody'
import { DashboardSkeleton } from '../components/DashboardSkeleton'
import { DashboardError } from '../components/DashboardError'
import { InviteUserLink } from '../components/InviteUserLink'

export default function SuperAdminDashboardPage() {
  const q = useSuperAdminDashboardQuery()
  return (
    <PageContainer>
      <PageHeader title="Dashboard" description="Organisation overview" actions={<InviteUserLink />} />
      {q.isPending ? (
        <DashboardSkeleton withOrganisation />
      ) : q.isError ? (
        <DashboardError error={q.error} onRetry={() => void q.refetch()} />
      ) : (
        <DashboardBody data={q.data} tenant={q.data.tenant} />
      )}
    </PageContainer>
  )
}
