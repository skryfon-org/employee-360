import { getErrorMessage } from '@employee360/api-client'

export function DashboardError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  return (
    <div role="alert" className="flex flex-col gap-3 rounded-sm border border-red-200 bg-red-50 p-3 text-sm text-red-700 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex gap-2">
        <svg aria-hidden="true" viewBox="0 0 24 24" width={20} height={20} fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" className="shrink-0">
          <circle cx="12" cy="12" r="9" />
          <path d="M12 8v5M12 16h.01" />
        </svg>
        <p>{getErrorMessage(error, 'Could not load the dashboard.')}</p>
      </div>
      <button
        type="button"
        onClick={onRetry}
        className="h-11 shrink-0 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
      >
        Retry
      </button>
    </div>
  )
}
