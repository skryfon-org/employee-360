import { useHolidayViewStore, type HolidayViewMode } from '../../../stores/holidayViewStore'

const OPTIONS: Array<{ mode: HolidayViewMode; label: string }> = [
  { mode: 'calendar', label: 'Calendar' },
  { mode: 'list', label: 'List' },
]

export function ViewToggle() {
  const viewMode = useHolidayViewStore((s) => s.viewMode)
  const setViewMode = useHolidayViewStore((s) => s.setViewMode)
  return (
    <div role="group" aria-label="View mode" className="inline-flex rounded-sm bg-slate-100 p-1">
      {OPTIONS.map(({ mode, label }) => (
        <button
          key={mode}
          type="button"
          aria-pressed={viewMode === mode}
          onClick={() => setViewMode(mode)}
          className={`h-9 rounded-sm px-3 text-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-7 ${
            viewMode === mode ? 'bg-white font-semibold text-slate-900' : 'font-medium text-slate-600 hover:text-slate-900'
          }`}
        >
          {label}
        </button>
      ))}
    </div>
  )
}
