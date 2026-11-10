import { useRef } from 'react'
import { useDialogFocus } from '../../../hooks/useDialogFocus'

export function ConfirmDeletePositionModal({
  name,
  loading,
  error,
  onConfirm,
  onCancel,
}: {
  name: string
  loading: boolean
  error?: string
  onConfirm: () => void
  onCancel: () => void
}) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const cancelRef = useRef<HTMLButtonElement>(null)
  useDialogFocus(dialogRef, cancelRef, onCancel, loading)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-position-title"
        aria-describedby="delete-position-desc"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-sm overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <div className="flex gap-3">
          <span
            aria-hidden="true"
            className="flex h-8 w-8 shrink-0 items-center justify-center rounded-sm border border-red-200 bg-red-50 text-red-700"
          >
            <svg viewBox="0 0 20 20" className="h-4 w-4" fill="currentColor">
              <path d="M10 2a8 8 0 100 16 8 8 0 000-16zm-1 4h2v5H9V6zm0 6h2v2H9v-2z" />
            </svg>
          </span>
          <div className="min-w-0">
            <h2 id="delete-position-title" className="text-base font-semibold text-slate-900">
              Delete position
            </h2>
            <p id="delete-position-desc" className="mt-1 text-sm text-slate-600">
              Delete <span className="break-words font-medium text-slate-900">{name}</span>? This cannot be undone.
            </p>
          </div>
        </div>
        {error && (
          <p role="alert" className="mt-3 rounded-sm border border-red-200 bg-red-50 p-2 text-xs text-red-700">
            {error}
          </p>
        )}
        <div className="mt-6 flex justify-end gap-2">
          <button
            ref={cancelRef}
            type="button"
            onClick={onCancel}
            disabled={loading}
            className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500 md:h-9"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={loading}
            aria-busy={loading}
            className="h-11 rounded-sm bg-red-600 px-4 text-sm font-semibold text-white hover:bg-red-700 active:bg-red-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-600 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-200 disabled:text-slate-500 md:h-9"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  )
}
