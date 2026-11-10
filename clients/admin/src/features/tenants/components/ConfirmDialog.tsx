import { useEffect, useRef, type ReactNode } from 'react'

const FOCUSABLE = 'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

interface Props {
  title: string
  children: ReactNode
  confirmLabel: string
  loading: boolean
  error?: string
  /** `danger` renders the solid red confirm button; `neutral` the primary one. */
  tone?: 'danger' | 'neutral'
  onConfirm: () => void
  onCancel: () => void
}

/** Accessible confirm modal: focus moves to Cancel, Tab is trapped, Escape cancels, focus is restored on close. */
export function ConfirmDialog({ title, children, confirmLabel, loading, error, tone = 'danger', onConfirm, onCancel }: Props) {
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
      if (items.length === 0) return e.preventDefault()
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

  const confirmCls =
    tone === 'danger'
      ? 'bg-red-600 hover:bg-red-700 active:bg-red-800 focus-visible:ring-red-600'
      : 'bg-slate-900 hover:bg-slate-800 active:bg-slate-950 focus-visible:ring-slate-900'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-sm overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <h2 id="confirm-title" className="text-base font-semibold text-slate-900">{title}</h2>
        <div className="mt-1 text-sm text-slate-600">{children}</div>
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
            className="h-11 md:h-9 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:text-slate-500"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={loading}
            aria-busy={loading}
            className={`h-11 md:h-9 rounded-sm px-4 text-sm font-semibold text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-200 disabled:text-slate-500 ${confirmCls}`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
