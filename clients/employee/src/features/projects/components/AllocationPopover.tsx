import { useEffect, useRef } from 'react'
import type { UserAllocation } from '../queries/projectQueries'
import type { AllocationState } from './projectDates'

const STATE_LABEL: Record<AllocationState, string> = { past: 'Past', active: 'Active', upcoming: 'Upcoming' }

interface Props {
  allocation: UserAllocation
  state: AllocationState
  anchor: DOMRect
  trigger: HTMLElement | null
  onClose: () => void
}

/** Small non-modal dialog anchored to a calendar bar; closes on Esc or outside click and restores focus. */
export function AllocationPopover({ allocation: a, state, anchor, trigger, onClose }: Props) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    ref.current?.focus()
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    const onDown = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node) && !trigger?.contains(e.target as Node)) onClose()
    }
    document.addEventListener('keydown', onKey)
    document.addEventListener('mousedown', onDown)
    return () => {
      document.removeEventListener('keydown', onKey)
      document.removeEventListener('mousedown', onDown)
      if (trigger?.isConnected) trigger.focus()
    }
  }, [onClose, trigger])

  const width = 288
  const left = Math.max(8, Math.min(anchor.left, window.innerWidth - width - 8))
  const top = anchor.bottom + 4

  return (
    <div
      ref={ref}
      role="dialog"
      aria-label={`${a.project_name} allocation details`}
      tabIndex={-1}
      style={{ position: 'fixed', top, left, width }}
      className="z-30 max-w-[calc(100vw-1rem)] rounded-sm border border-slate-300 bg-white p-4 text-sm text-slate-900 shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900"
    >
      <div className="flex items-start justify-between gap-2">
        <div>
          <p className="font-semibold">{a.project_name}</p>
          <p className="text-xs text-slate-600">{a.project_code}</p>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close details"
          className="h-9 w-9 shrink-0 rounded-sm text-slate-600 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900"
        >
          <span aria-hidden="true">x</span>
        </button>
      </div>
      <dl className="mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
        <dt className="text-slate-600">Allocation</dt>
        <dd>{a.allocation_percent}%</dd>
        <dt className="text-slate-600">Role</dt>
        <dd>{a.role_on_project || '-'}</dd>
        <dt className="text-slate-600">Start</dt>
        <dd>{a.start_date}</dd>
        <dt className="text-slate-600">End</dt>
        <dd>{a.end_date ?? 'Ongoing'}</dd>
        <dt className="text-slate-600">State</dt>
        <dd>{STATE_LABEL[state]}</dd>
      </dl>
    </div>
  )
}
