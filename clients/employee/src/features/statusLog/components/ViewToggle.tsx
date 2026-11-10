export type StatusView = 'week' | 'calendar'

const options: { value: StatusView; label: string }[] = [
  { value: 'week', label: 'Week' },
  { value: 'calendar', label: 'Calendar' },
]

/** Segmented Week / Calendar switch (tablist; Left/Right move between the tabs). */
export function ViewToggle({ view, onChange, panelId }: { view: StatusView; onChange: (v: StatusView) => void; panelId: string }) {
  return (
    <div role="tablist" aria-label="Status view" className="inline-flex rounded-sm bg-slate-100 p-1">
      {options.map((o) => (
        <button
          key={o.value}
          id={`${panelId}-tab-${o.value}`}
          type="button"
          role="tab"
          aria-selected={view === o.value}
          aria-controls={panelId}
          tabIndex={view === o.value ? 0 : -1}
          onClick={() => onChange(o.value)}
          onKeyDown={(e) => {
            if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return
            e.preventDefault()
            const next = options[(options.findIndex((x) => x.value === o.value) + (e.key === 'ArrowRight' ? 1 : -1) + options.length) % options.length]
            onChange(next.value)
            document.getElementById(`${panelId}-tab-${next.value}`)?.focus()
          }}
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
