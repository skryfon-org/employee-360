import { ALLOCATION_STATE_LABELS, ALLOCATION_STATE_STYLES, type AllocationState } from '../allocationState'

export function AllocationStateBadge({ state }: { state: AllocationState }) {
  const s = ALLOCATION_STATE_STYLES[state]
  return (
    <span className={`inline-flex h-6 items-center gap-1.5 rounded-sm border px-2 text-xs font-medium ${s.box}`}>
      <span aria-hidden="true" className={`h-1.5 w-1.5 rounded-full ${s.dot}`} />
      {ALLOCATION_STATE_LABELS[state]}
    </span>
  )
}
