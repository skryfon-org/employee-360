import { useState } from 'react'
import { PageContainer, PageHeader } from '@employee360/ui'
import { DailyView } from '../components/DailyView'
import { MonthlyView } from '../components/MonthlyView'

type View = 'daily' | 'monthly'
const VIEWS: { id: View; label: string }[] = [
  { id: 'daily', label: 'Daily' },
  { id: 'monthly', label: 'Monthly' },
]

/** Admin status monitoring: a daily view (who logged on a date) and a monthly org-wide report. */
export default function StatusLogsPage() {
  const [view, setView] = useState<View>('daily')
  return (
    <PageContainer>
      <PageHeader
        title="Status logs"
        description="See who has logged their daily status, by day or for the whole month."
        actions={
          <div role="group" aria-label="View" className="inline-flex rounded-sm bg-slate-100 p-1">
            {VIEWS.map((v) => (
              <button
                key={v.id}
                type="button"
                aria-pressed={view === v.id}
                onClick={() => setView(v.id)}
                className={`h-9 rounded-sm px-4 text-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 ${
                  view === v.id ? 'bg-white font-semibold text-slate-900' : 'font-medium text-slate-600 hover:text-slate-900'
                }`}
              >
                {v.label}
              </button>
            ))}
          </div>
        }
      />
      {view === 'daily' ? <DailyView /> : <MonthlyView />}
    </PageContainer>
  )
}
