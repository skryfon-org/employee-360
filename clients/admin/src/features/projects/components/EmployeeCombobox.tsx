import { useEffect, useId, useRef, useState } from 'react'
import { getErrorMessage } from '@employee360/api-client'
import { useEmployeeOptionsQuery, type EmployeeSummary } from '../queries/projectQueries'
import { fullName } from '../../employees/employeeName'
import { useDebouncedValue } from '../../../hooks/useDebouncedValue'

const INPUT_CLS =
  'h-11 w-full rounded-sm border border-slate-300 bg-white px-3 pr-16 text-base text-slate-900 placeholder:text-slate-500 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 aria-[invalid=true]:focus:ring-red-600 sm:text-sm md:h-9'

const RESULT_LIMIT = 10

const labelOf = (e: EmployeeSummary) => (e.employee_code ? `${fullName(e)} (${e.employee_code})` : fullName(e))

/** Text + colour hint for how much of someone's time is already allocated today (warn-only). */
export function allocationHint(percent: number | undefined): { text: string; warn: boolean } | null {
  if (percent === undefined) return null
  if (percent > 100) return { text: `Over-allocated (${percent}% allocated)`, warn: true }
  if (percent === 100) return { text: 'Fully allocated (100% allocated)', warn: true }
  if (percent === 0) return { text: 'Not allocated', warn: false }
  return { text: `${percent}% allocated`, warn: false }
}

interface Props {
  label: string
  selected: EmployeeSummary | null
  onSelect: (employee: EmployeeSummary | null) => void
  error?: string
  required?: boolean
  /** Only search active employees (default true; leave filters need inactive people too). */
  activeOnly?: boolean
  /** Request and show allocation hints (default true; only the allocation form wants them). */
  showAllocation?: boolean
}

/**
 * Single search combobox for employees (ARIA 1.2 combobox + listbox). Typing searches the server by
 * name, email and employee code (debounced); picking a result selects it, editing the text clears it.
 */
export function EmployeeCombobox({ label, selected, onSelect, error, required, activeOnly = true, showAllocation = true }: Props) {
  const id = useId()
  const listId = `${id}-list`
  const rootRef = useRef<HTMLDivElement>(null)
  const [text, setText] = useState(selected ? labelOf(selected) : '')
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(0)
  const debounced = useDebouncedValue(selected ? '' : text.trim(), 300)
  const query = useEmployeeOptionsQuery(debounced, activeOnly, RESULT_LIMIT, open && !selected, showAllocation)
  const results = selected ? [] : (query.data ?? [])
  // Results lag the typed text by the debounce; treat that gap as loading too.
  const loading = open && !selected && (query.isFetching || debounced !== text.trim())

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [open])

  const pick = (e: EmployeeSummary) => {
    onSelect(e)
    setText(labelOf(e))
    setOpen(false)
  }
  const clear = () => {
    onSelect(null)
    setText('')
    setActive(0)
    setOpen(true)
  }

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      if (!open) setOpen(true)
      else setActive((a) => Math.min(a + 1, results.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActive((a) => Math.max(a - 1, 0))
    } else if (e.key === 'Enter' && open) {
      e.preventDefault()
      if (results[active]) pick(results[active])
    } else if (e.key === 'Escape' && open) {
      // Keep Escape from also closing the surrounding dialog.
      e.preventDefault()
      e.stopPropagation()
      setOpen(false)
    }
  }

  const activeId = open && results[active] ? `${id}-opt-${active}` : undefined
  const errId = `${id}-error`
  return (
    <div ref={rootRef} className="relative flex flex-col gap-1">
      <label htmlFor={id} className="text-xs font-medium text-slate-900">
        {label}
        {required && (
          <span className="text-red-600" aria-hidden="true">
            {' '}
            *
          </span>
        )}
      </label>
      <div className="relative">
        <input
          id={id}
          role="combobox"
          type="text"
          autoComplete="off"
          aria-expanded={open}
          aria-controls={listId}
          aria-autocomplete="list"
          aria-activedescendant={activeId}
          aria-required={required || undefined}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? errId : undefined}
          placeholder="Search by name, email or employee code"
          className={INPUT_CLS}
          value={text}
          onFocus={() => setOpen(true)}
          onKeyDown={onKeyDown}
          onChange={(e) => {
            setText(e.target.value)
            setActive(0)
            setOpen(true)
            if (selected) onSelect(null)
          }}
        />
        <div className="absolute inset-y-0 right-0 flex items-center gap-1 pr-2">
          {loading && (
            <svg className="h-4 w-4 animate-spin text-slate-600" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="3" opacity="0.25" />
              <path d="M21 12a9 9 0 0 0-9-9" stroke="currentColor" strokeWidth="3" />
            </svg>
          )}
          {text && (
            <button
              type="button"
              aria-label="Clear employee"
              onClick={clear}
              className="flex h-6 w-6 items-center justify-center rounded-sm text-slate-600 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900"
            >
              <svg viewBox="0 0 20 20" className="h-4 w-4" fill="currentColor" aria-hidden="true">
                <path d="M6.3 5 5 6.3 8.7 10 5 13.7 6.3 15 10 11.3 13.7 15 15 13.7 11.3 10 15 6.3 13.7 5 10 8.7z" />
              </svg>
            </button>
          )}
        </div>
      </div>
      {open && !selected && (
        <ul
          id={listId}
          role="listbox"
          aria-label={`${label} results`}
          className="absolute left-0 right-0 top-full z-10 mt-1 max-h-60 overflow-y-auto rounded-sm border border-slate-300 bg-white"
        >
          {results.map((e, i) => (
            <li
              key={e.id}
              id={`${id}-opt-${i}`}
              role="option"
              aria-selected={i === active}
              onMouseDown={(ev) => ev.preventDefault()}
              onMouseEnter={() => setActive(i)}
              onClick={() => pick(e)}
              className={`cursor-pointer px-3 py-2 ${i === active ? 'bg-slate-200' : 'hover:bg-slate-100'}`}
            >
              <p className="text-sm font-medium text-slate-900">{fullName(e)}</p>
              <p className="text-xs text-slate-600">
                {[e.employee_code, e.email, e.department_name].filter(Boolean).join(' · ')}
              </p>
              {(() => {
                const hint = showAllocation ? allocationHint(e.allocated_percent) : null
                return hint && <p className={`text-xs ${hint.warn ? 'font-medium text-amber-800' : 'text-slate-600'}`}>{hint.text}</p>
              })()}
            </li>
          ))}
          {query.isError && (
            <li role="presentation" className="px-3 py-2 text-sm text-red-700">
              {getErrorMessage(query.error, 'Could not search employees.')}
            </li>
          )}
          {!query.isError && !loading && results.length === 0 && (
            <li role="presentation" className="px-3 py-2 text-sm text-slate-600">
              No matching employees
            </li>
          )}
          {!query.isError && loading && results.length === 0 && (
            <li role="presentation" className="px-3 py-2 text-sm text-slate-600">
              Searching…
            </li>
          )}
        </ul>
      )}
      {error && (
        <p id={errId} role="alert" className="text-xs text-red-700">
          {error}
        </p>
      )}
    </div>
  )
}
