import { useEffect, useRef } from 'react'

const FOCUSABLE = 'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

export function ConfirmRevokeModal({
  email,
  loading,
  error,
  onConfirm,
  onCancel,
}: {
  email: string
  loading: boolean
  error?: string
  onConfirm: () => void
  onCancel: () => void
}) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const cancelRef = useRef<HTMLButtonElement>(null)
  const loadingRef = useRef(loading)
  const onCancelRef = useRef(onCancel)
  useEffect(() => {
    loadingRef.current = loading
    onCancelRef.current = onCancel
  })

  useEffect(() => {
    const trigger = document.activeElement instanceof HTMLElement ? document.activeElement : null
    cancelRef.current?.focus()

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        if (!loadingRef.current) onCancelRef.current()
        return
      }
      if (e.key !== 'Tab') return
      const items = Array.from(dialogRef.current?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? [])
      if (items.length === 0) {
        e.preventDefault()
        return
      }
      const first = items[0]
      const last = items[items.length - 1]
      const active = document.activeElement
      if (!dialogRef.current?.contains(active)) {
        e.preventDefault()
        first.focus()
      } else if (e.shiftKey && active === first) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && active === last) {
        e.preventDefault()
        first.focus()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      if (trigger?.isConnected) trigger.focus()
    }
  }, [])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="revoke-title"
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
            <h2 id="revoke-title" className="text-base font-semibold text-slate-900">
              Revoke invitation
            </h2>
            <p className="mt-1 text-sm text-slate-600">
              Revoke the invitation for <span className="break-all font-medium text-slate-900">{email}</span>? They will no
              longer be able to accept it.
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
            className="h-11 md:h-9 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={loading}
            aria-busy={loading}
            className="h-11 md:h-9 rounded-sm bg-red-600 px-4 text-sm font-semibold text-white hover:bg-red-700 active:bg-red-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-600 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-200 disabled:text-slate-500"
          >
            Revoke
          </button>
        </div>
      </div>
    </div>
  )
}
