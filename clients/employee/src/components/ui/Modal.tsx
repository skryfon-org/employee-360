import { useEffect, useId, useRef, type ReactNode } from 'react'

const FOCUSABLE = 'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

/** Open modals, oldest first; only the topmost handles keyboard events (supports a confirm dialog over a form dialog). */
const openModals: symbol[] = []
let savedOverflow = ''

/**
 * Accessible modal dialog: labelled by `title`, focus moves in and is trapped, Escape/backdrop close,
 * focus returns to the trigger on close, body scroll is locked while open. Mount it only while open.
 */
export function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  const titleId = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  const onCloseRef = useRef(onClose)
  useEffect(() => {
    onCloseRef.current = onClose
  })

  useEffect(() => {
    const trigger = document.activeElement instanceof HTMLElement ? document.activeElement : null
    const me = Symbol('modal')
    if (openModals.length === 0) savedOverflow = document.body.style.overflow
    openModals.push(me)
    document.body.style.overflow = 'hidden'
    const first = dialogRef.current?.querySelector<HTMLElement>(FOCUSABLE)
    ;(first ?? dialogRef.current)?.focus()

    const onKeyDown = (e: KeyboardEvent) => {
      if (openModals[openModals.length - 1] !== me) return
      if (e.key === 'Escape') {
        e.preventDefault()
        onCloseRef.current()
        return
      }
      if (e.key !== 'Tab') return
      const items = Array.from(dialogRef.current?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? [])
      if (items.length === 0) return e.preventDefault()
      const firstItem = items[0]
      const last = items[items.length - 1]
      const active = document.activeElement
      if (!dialogRef.current?.contains(active) || (e.shiftKey && active === firstItem) || (!e.shiftKey && active === last)) {
        e.preventDefault()
        ;(e.shiftKey ? last : firstItem).focus()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      openModals.splice(openModals.indexOf(me), 1)
      if (openModals.length === 0) document.body.style.overflow = savedOverflow
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
        aria-labelledby={titleId}
        tabIndex={-1}
        className="max-h-[calc(100dvh-2rem)] w-full max-w-lg overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 focus:outline-none sm:p-6"
      >
        <h2 id={titleId} className="mb-4 text-base font-semibold text-slate-900">
          {title}
        </h2>
        {children}
      </div>
    </div>
  )
}
