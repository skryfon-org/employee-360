import { useCallback, useEffect, useId, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'

const INPUT_CLS =
  'h-11 w-full rounded-sm border border-slate-300 bg-white px-3 pr-9 text-base text-slate-900 placeholder:text-slate-500 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500 aria-[invalid=true]:border-red-600 sm:text-sm md:h-9'

/** Most options rendered at once; the rest are reached by typing to narrow the list. */
export const COMBOBOX_MAX_RENDERED = 100
const MAX_LIST_HEIGHT = 240 // 15rem
const GAP = 4

export interface ComboboxProps {
  id: string
  label: string
  /** Currently selected option (or ''). */
  value: string
  options: string[]
  onChange: (value: string) => void
  onBlur?: () => void
  placeholder?: string
  required?: boolean
  error?: string
  disabled?: boolean
}

/** Prefix matches first, then other substring matches; stable within each group. */
export function filterOptions(options: string[], query: string): string[] {
  const q = query.trim().toLocaleLowerCase()
  if (!q) return options
  const starts: string[] = []
  const contains: string[] = []
  for (const o of options) {
    const i = o.toLocaleLowerCase().indexOf(q)
    if (i === 0) starts.push(o)
    else if (i > 0) contains.push(o)
  }
  return [...starts, ...contains]
}

/**
 * Searchable single-select combobox (ARIA 1.2 combobox + listbox). The list is
 * portalled with fixed positioning so cards and modals never clip it; it opens
 * below the input at the input's width and flips above only when it does not fit
 * below and there is more room above.
 */
export function Combobox({ id, label, value, options, onChange, onBlur, placeholder, required, error, disabled }: ComboboxProps) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [typed, setTyped] = useState(false)
  const [active, setActive] = useState(0)
  const [pos, setPos] = useState<{ top: number; left: number; width: number } | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLUListElement>(null)
  const listId = useId()

  const matches = useMemo(() => filterOptions(options, typed ? query : ''), [options, query, typed])
  const shown = matches.slice(0, COMBOBOX_MAX_RENDERED)
  const truncated = matches.length > shown.length

  const place = useCallback(() => {
    const input = inputRef.current
    const list = listRef.current
    if (!input || !list) return
    const r = input.getBoundingClientRect()
    const height = Math.min(list.getBoundingClientRect().height, MAX_LIST_HEIGHT)
    const spaceBelow = window.innerHeight - r.bottom - GAP
    const spaceAbove = r.top - GAP
    const flipUp = height > spaceBelow && spaceAbove > spaceBelow
    const top = flipUp ? Math.max(GAP, r.top - GAP - height) : r.bottom + GAP
    setPos({ top, left: r.left, width: r.width })
  }, [])

  // Measure after mount (the list renders hidden at 0,0) and whenever its content changes.
  useLayoutEffect(() => {
    if (open) place()
  }, [open, shown.length, truncated, place])

  useEffect(() => {
    if (!open) return
    window.addEventListener('resize', place)
    window.addEventListener('scroll', place, true)
    return () => {
      window.removeEventListener('resize', place)
      window.removeEventListener('scroll', place, true)
    }
  }, [open, place])

  const openList = () => {
    if (disabled || open) return
    setTyped(false)
    setQuery('')
    const idx = options.indexOf(value)
    setActive(idx >= 0 && idx < COMBOBOX_MAX_RENDERED ? idx : 0)
    setPos(null)
    setOpen(true)
  }
  const close = () => {
    setOpen(false)
    setTyped(false)
    setQuery('')
  }
  const select = (v: string) => {
    onChange(v)
    close()
  }

  // Keep the active option in view.
  useEffect(() => {
    if (open && pos) listRef.current?.querySelector(`[id="${listId}-${active}"]`)?.scrollIntoView?.({ block: 'nearest' })
  }, [open, active, pos, listId])

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        if (!open) openList()
        else setActive((a) => Math.min(a + 1, shown.length - 1))
        break
      case 'ArrowUp':
        e.preventDefault()
        if (!open) openList()
        else setActive((a) => Math.max(a - 1, 0))
        break
      case 'Home':
        if (open) {
          e.preventDefault()
          setActive(0)
        }
        break
      case 'End':
        if (open) {
          e.preventDefault()
          setActive(Math.max(shown.length - 1, 0))
        }
        break
      case 'Enter':
        if (open) {
          e.preventDefault()
          if (shown[active] !== undefined) select(shown[active])
        }
        break
      case 'Escape':
        if (open) {
          e.preventDefault()
          close()
        }
        break
      case 'Tab':
        if (open) {
          // Commit the highlighted option only when the user narrowed the list; emptied text clears.
          if (typed && query.trim() === '') {
            if (value !== '') onChange('')
          } else if (typed && shown[active] !== undefined) onChange(shown[active])
          close()
        }
        break
    }
  }

  const handleBlur = () => {
    if (open) {
      // Emptying the text and leaving clears the selection.
      if (typed && query.trim() === '' && value !== '') onChange('')
      close()
    }
    onBlur?.()
  }

  const activeId = open && shown[active] !== undefined ? `${listId}-${active}` : undefined
  const errId = `${id}-error`

  return (
    <div className="flex flex-col gap-1">
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
          ref={inputRef}
          id={id}
          name={id}
          type="text"
          role="combobox"
          autoComplete="off"
          aria-expanded={open}
          aria-controls={open ? listId : undefined}
          aria-activedescendant={activeId}
          aria-autocomplete="list"
          aria-required={required || undefined}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? errId : undefined}
          disabled={disabled}
          placeholder={placeholder}
          className={INPUT_CLS}
          value={open && typed ? query : value}
          onChange={(e) => {
            if (!open) setOpen(true)
            setTyped(true)
            setQuery(e.target.value)
            setActive(0)
          }}
          onFocus={(e) => e.currentTarget.select()}
          onClick={(e) => {
            if (!open) openList()
            e.currentTarget.select()
          }}
          onKeyDown={onKeyDown}
          onBlur={handleBlur}
        />
        <svg aria-hidden="true" viewBox="0 0 20 20" className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" fill="none" stroke="currentColor" strokeWidth="1.5">
          <path d="M5 8l5 5 5-5" />
        </svg>
      </div>
      {open &&
        createPortal(
          <ul
            ref={listRef}
            id={listId}
            role="listbox"
            aria-label={label}
            // Keep focus in the input while interacting with the list.
            onMouseDown={(e) => e.preventDefault()}
            style={{
              position: 'fixed',
              top: pos?.top ?? 0,
              left: pos?.left ?? 0,
              width: pos?.width,
              maxHeight: MAX_LIST_HEIGHT,
              visibility: pos ? 'visible' : 'hidden',
            }}
            className="z-50 overflow-y-auto rounded-sm border border-slate-200 bg-white py-1 shadow-md"
          >
            {shown.length === 0 && <li role="presentation" className="px-3 py-2 text-sm text-slate-600">No matches</li>}
            {shown.map((o, i) => (
              <li
                key={o}
                id={`${listId}-${i}`}
                role="option"
                aria-selected={o === value}
                onMouseEnter={() => setActive(i)}
                onClick={() => select(o)}
                className={`flex min-h-9 cursor-pointer items-center px-3 py-1.5 text-sm text-slate-900 ${
                  i === active ? 'bg-slate-100' : ''
                } ${o === value ? 'font-semibold' : ''}`}
              >
                {o}
              </li>
            ))}
            {truncated && (
              <li role="presentation" className="px-3 py-2 text-xs text-slate-600">
                Showing first {COMBOBOX_MAX_RENDERED} of {matches.length}. Type to narrow results.
              </li>
            )}
          </ul>,
          document.body,
        )}
      {error && (
        <p id={errId} role="alert" className="text-xs text-red-700">
          {error}
        </p>
      )}
    </div>
  )
}
