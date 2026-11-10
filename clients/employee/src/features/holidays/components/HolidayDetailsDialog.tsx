import { useEffect, useRef } from 'react'
import type { Holiday } from '../queries/holidayQueries'
import { formatHolidayDate } from '../dates'
import { CategoryBadge, type CategoryStatus } from './CategoryBadge'

const FOCUSABLE = 'button:not([disabled]), [href], [tabindex]:not([tabindex="-1"])'

/** Read-only holiday details modal: focus moves in, Tab is trapped, Escape/backdrop/Close dismiss, focus is restored. */
export function HolidayDetailsDialog({
  holiday,
  categoryStatus = 'ready',
  onClose,
}: {
  holiday: Holiday
  categoryStatus?: CategoryStatus
  onClose: () => void
}) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  const onCloseRef = useRef(onClose)
  useEffect(() => {
    onCloseRef.current = onClose
  })

  useEffect(() => {
    const trigger = document.activeElement instanceof HTMLElement ? document.activeElement : null
    closeRef.current?.focus()
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onCloseRef.current()
        return
      }
      if (e.key !== 'Tab') return
      const items = Array.from(dialogRef.current?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? [])
      if (items.length === 0) return e.preventDefault()
      const first = items[0]
      const last = items[items.length - 1]
      const active = document.activeElement
      if (!dialogRef.current?.contains(active) || (e.shiftKey && active === first) || (!e.shiftKey && active === last)) {
        e.preventDefault()
        ;(e.shiftKey && active === first ? last : first).focus()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      if (trigger?.isConnected) trigger.focus()
    }
  }, [])

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="holiday-details-title"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-sm overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <h2 id="holiday-details-title" className="break-words text-base font-semibold text-slate-900">
          {holiday.name}
        </h2>
        <p className="mt-1 text-sm text-slate-600">{formatHolidayDate(holiday.date)}</p>
        <div className="mt-3">
          <CategoryBadge category={holiday.category} status={categoryStatus} />
        </div>
        {holiday.description && <p className="mt-3 whitespace-pre-line break-words text-sm text-slate-900">{holiday.description}</p>}
        <div className="mt-6 flex justify-end">
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
