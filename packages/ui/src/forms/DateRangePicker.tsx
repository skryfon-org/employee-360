import { useEffect, useId, useState } from 'react'
import DatePicker from 'react-datepicker'
import 'react-datepicker/dist/react-datepicker.css'
import './dateRangePicker.css'

export interface DateRangeValue {
  /** YYYY-MM-DD, inclusive; empty string = unset. */
  from: string
  to: string
}

export interface DateRangePickerProps {
  value: DateRangeValue
  onChange: (value: DateRangeValue) => void
  label?: string
  id?: string
}

/** Local-time parse (never `new Date('YYYY-MM-DD')`, which is UTC and can shift the day). */
function parse(iso: string): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso)
  return m ? new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : null
}

/** Local-time format (never `toISOString`, which converts to UTC). */
function toIso(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

const ICON = (
  <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.75} strokeLinecap="round" strokeLinejoin="round">
    <rect x="3" y="5" width="18" height="16" rx="2" />
    <path d="M3 10h18M8 3v4M16 3v4" />
  </svg>
)

/**
 * Single-input date range selector (react-datepicker, `selectsRange`) with a built-in popover.
 * `onChange` fires only for a complete range (a single day gives from === to) or when cleared,
 * so list queries are not triggered by a half-picked range.
 */
export function DateRangePicker({ value, onChange, label = 'Date range', id }: DateRangePickerProps) {
  const autoId = useId()
  const inputId = id ?? `${autoId}-range`
  const [draft, setDraft] = useState<[Date | null, Date | null]>([parse(value.from), parse(value.to)])

  // Follow external changes (URL, Clear button).
  useEffect(() => {
    setDraft([parse(value.from), parse(value.to)])
  }, [value.from, value.to])

  return (
    <div className="e360-drp flex w-full flex-col gap-1 sm:w-64">
      <label htmlFor={inputId} className="text-xs font-medium text-slate-900">
        {label}
      </label>
      <DatePicker
        id={inputId}
        selectsRange
        startDate={draft[0]}
        endDate={draft[1]}
        onChange={(range) => {
          const [s, e] = range
          setDraft([s, e])
          if (!s && !e) onChange({ from: '', to: '' })
          else if (s && e) onChange({ from: toIso(s), to: toIso(e) })
        }}
        dateFormat="dd MMM yyyy"
        placeholderText="Select date range"
        isClearable
        ariaLabelClose="Clear date range"
        showIcon
        icon={ICON}
        autoComplete="off"
        className="h-11 w-full rounded-sm border border-slate-300 bg-white text-base text-slate-900 placeholder:text-slate-600 hover:bg-slate-50 focus:border-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9"
      />
    </div>
  )
}
