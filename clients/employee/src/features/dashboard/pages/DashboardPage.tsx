import { getErrorMessage } from '@employee360/api-client'
import { PageContainer } from '@employee360/ui'
import { useEmployeeDashboardQuery } from '../queries/dashboardQueries'
import { DashboardHeader } from '../components/DashboardHeader'
import { DashboardSkeleton } from '../components/DashboardSkeleton'
import { KpiTiles } from '../components/KpiTiles'
import { LeaveOverviewCard } from '../components/LeaveOverviewCard'
import { HolidaysCard } from '../components/HolidaysCard'
import { ProjectsCard } from '../components/ProjectsCard'
import { WeekCard } from '../components/WeekCard'

export default function DashboardPage() {
  const { data, isPending, isError, error, refetch } = useEmployeeDashboardQuery()

  return (
    <PageContainer>
      {isPending && <DashboardSkeleton />}
      {isError && (
        <div role="alert" className="flex flex-col gap-3 rounded-sm border border-red-200 bg-red-50 p-3 text-sm text-red-700 sm:flex-row sm:items-center sm:justify-between">
          <p>{getErrorMessage(error, 'Could not load your dashboard.')}</p>
          <button
            type="button"
            onClick={() => void refetch()}
            className="h-9 shrink-0 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
          >
            Retry
          </button>
        </div>
      )}
      {data && (
        <>
          <DashboardHeader greeting={data.greeting} />
          <KpiTiles stats={data.stats} />
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <LeaveOverviewCard leave={data.leave} />
            <HolidaysCard holidays={data.holidays} />
            <ProjectsCard projects={data.projects} />
            <WeekCard week={data.week} />
          </div>
        </>
      )}
    </PageContainer>
  )
}
