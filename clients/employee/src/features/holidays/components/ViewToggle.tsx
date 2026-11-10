import type { HolidayViewMode } from '../../../stores/uiStore'

const options: { value: HolidayViewMode; label: string }[] = [
  { value: 'calendar', label: 'Calendar' },
  { value: 'list', label: 'List' },
]

export function ViewToggle({ view, onChange }: { view: HolidayViewMode; onChange: (v: HolidayViewMode) => void }) {
  return (
    <div role="group" aria-label="Holiday view" className="inline-flex rounded-sm bg-slate-100 p-1">
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          aria-pressed={view === o.value}
          onClick={() => onChange(o.value)}
          className={`h-9 rounded-sm px-3 text-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-7 ${
            view === o.value ? 'bg-white font-semibold text-slate-900' : 'font-medium text-slate-600 hover:text-slate-900'
          }`}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}
