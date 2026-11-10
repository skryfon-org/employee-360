export type ProjectsViewMode = 'list' | 'calendar'

const options: { value: ProjectsViewMode; label: string }[] = [
  { value: 'list', label: 'List' },
  { value: 'calendar', label: 'Calendar' },
]

export function ProjectsViewToggle({ view, onChange }: { view: ProjectsViewMode; onChange: (v: ProjectsViewMode) => void }) {
  return (
    <div role="group" aria-label="Projects view" className="inline-flex rounded-sm bg-slate-100 p-1">
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
