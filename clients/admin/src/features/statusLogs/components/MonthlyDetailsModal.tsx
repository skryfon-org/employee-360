import { useRef } from 'react'
import { useDialogFocus } from '../../../hooks/useDialogFocus'

export interface MonthlyDetails {
  name: string
  logged: number
  expected: number
  missing: number
  leave: number
  offDays: number
}

/** Per-employee month breakdown opened from the info icon in the "Logged / expected" cell. */
export function MonthlyDetailsModal({ details, onClose }: { details: MonthlyDetails; onClose: () => void }) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  useDialogFocus(dialogRef, closeRef, onClose, false)

  const rows: Array<{ label: string; value: string; warn?: boolean }> = [
    { label: 'Logged', value: `${details.logged}/${details.expected}` },
    { label: 'Missing', value: String(details.missing), warn: details.missing > 0 },
    { label: 'On leave', value: String(details.leave) },
    { label: 'Logged on off-days', value: String(details.offDays) },
  ]

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="monthly-details-title"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-sm overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <h2 id="monthly-details-title" className="break-words text-base font-semibold text-slate-900">
          {details.name}
        </h2>
        <p className="mt-1 text-xs text-slate-600">
          Expected days exclude weekly offs, public holidays and approved leave.
        </p>
        <dl className="mt-4 divide-y divide-slate-200 text-sm">
          {rows.map((r) => (
            <div key={r.label} className="flex items-center justify-between py-2">
              <dt className="text-slate-600">{r.label}</dt>
              <dd className={r.warn ? 'font-semibold text-amber-700' : 'font-medium text-slate-900'}>{r.value}</dd>
            </div>
          ))}
        </dl>
        <div className="mt-4 flex justify-end">
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-900 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
