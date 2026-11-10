import { useId } from 'react'
import type { ComponentType, SVGProps } from 'react'
import { Link } from 'react-router-dom'

function ArrowRightIcon({ className }: { className?: string }) {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" width={16} height={16} fill="none" stroke="currentColor" strokeWidth={1.75} strokeLinecap="round" strokeLinejoin="round" className={className}>
      <path d="M5 12h14M13 6l6 6-6 6" />
    </svg>
  )
}

export interface Stat {
  label: string
  value: number | string | undefined
  description: string
  icon: ComponentType<SVGProps<SVGSVGElement>>
  /** When set, the whole card links to this route and shows a subtle arrow. */
  to?: string
}

const CARD = 'flex h-full min-w-0 flex-col gap-1 rounded-sm border border-line bg-surface p-4'

function StatCard({ stat }: { stat: Stat }) {
  const Icon = stat.icon
  const id = useId()
  const labelId = `${id}-label`
  const valueId = `${id}-value`
  const descId = `${id}-desc`
  const content = (
    <>
      <div className="mb-2 flex items-center justify-between text-slate-600">
        <Icon width={18} height={18} className="text-slate-700" />
        {stat.to && <ArrowRightIcon className="text-slate-600 transition-colors group-hover:text-slate-900" />}
      </div>
      <p id={labelId} className="text-xs font-medium text-slate-600">{stat.label}</p>
      <p id={valueId} className="text-2xl font-semibold text-slate-900">{stat.value ?? 0}</p>
      <p id={descId} className="text-xs text-slate-600">{stat.description}</p>
    </>
  )
  return stat.to ? (
    <Link
      to={stat.to}
      aria-labelledby={`${labelId} ${valueId}`}
      aria-describedby={descId}
      className={`group ${CARD} hover:border-slate-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2`}
    >
      {content}
    </Link>
  ) : (
    <div role="group" aria-labelledby={labelId} aria-describedby={`${valueId} ${descId}`} className={CARD}>
      {content}
    </div>
  )
}

const COLUMNS: Record<2 | 4, string> = {
  2: 'sm:grid-cols-2',
  4: 'sm:grid-cols-2 lg:grid-cols-4',
}

/** Reusable KPI card grid. `columns` is the desktop column count (always 1 on mobile, 2 on tablet). */
export function StatCards({ stats, columns = 4, label }: { stats: Stat[]; columns?: 2 | 4; label: string }) {
  return (
    <ul
      aria-label={label}
      className={`grid grid-cols-1 gap-4 ${COLUMNS[columns]}`}
    >
      {stats.map((s) => (
        <li key={s.label} className="min-w-0">
          <StatCard stat={s} />
        </li>
      ))}
    </ul>
  )
}
