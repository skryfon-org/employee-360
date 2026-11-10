import { FilterBar } from '@employee360/ui'
import { useEffect, useState } from 'react'
import { useDebouncedValue } from '../../../hooks/useDebouncedValue'
import { INVITATION_STATUSES, SEARCH_MAX_LENGTH } from '../schemas/invitationListSchema'

const FIELD =
  'h-11 md:h-9 rounded-sm border border-slate-300 bg-white px-3 text-sm text-slate-900 placeholder:text-slate-500 focus:outline-none focus:border-slate-900 focus:ring-1 focus:ring-slate-900'

interface Props {
  status?: string
  search?: string
  onStatusChange: (status: string | undefined) => void
  onSearchChange: (search: string | undefined) => void
}

export function InvitationsToolbar({ status, search, onStatusChange, onSearchChange }: Props) {
  const [text, setText] = useState(search ?? '')
  const debounced = useDebouncedValue(text.trim(), 300)

  useEffect(() => {
    if (debounced !== (search ?? '')) onSearchChange(debounced || undefined)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debounced])

  return (
    <FilterBar className="flex flex-col gap-3 sm:flex-row sm:items-end">
      <div className="flex flex-1 flex-col gap-1">
        <label htmlFor="inv-search" className="text-xs font-medium text-slate-900">Search</label>
        <input
          id="inv-search"
          type="search"
          value={text}
          maxLength={SEARCH_MAX_LENGTH}
          onChange={(e) => setText(e.target.value)}
          placeholder="Search by invitee email"
          className={FIELD}
        />
      </div>
      <div className="flex flex-col gap-1">
        <label htmlFor="inv-status" className="text-xs font-medium text-slate-900">Status</label>
        <select
          id="inv-status"
          value={status ?? ''}
          onChange={(e) => onStatusChange(e.target.value || undefined)}
          className={FIELD}
        >
          <option value="">All</option>
          {INVITATION_STATUSES.map((s) => (
            <option key={s} value={s}>{s.charAt(0).toUpperCase() + s.slice(1)}</option>
          ))}
        </select>
      </div>
    </FilterBar>
  )
}
