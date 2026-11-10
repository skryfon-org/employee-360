import { useCallback, useEffect, useId, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'

export interface RowAction {
  label: string
  onSelect: () => void
  /** Destructive action, rendered in red. */
  danger?: boolean
  disabled?: boolean
  /** Native tooltip, e.g. to explain why the action is disabled. */
  title?: string
}

export interface RowActionsMenuProps {
  /** Human name of the row, used for the trigger's accessible label. */
  label: string
  actions: RowAction[]
}

const MENU_WIDTH = 176

/**
 * Three-dot row actions menu. The menu is portalled and fixed-positioned so it
 * is never clipped by table overflow containers.
 */
export function RowActionsMenu({ label, actions }: RowActionsMenuProps) {
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  const enabledItems = useCallback(
    () => Array.from(menuRef.current?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]:not(:disabled)') ?? []),
    [],
  )

  const close = useCallback((restoreFocus = true) => {
    setOpen(false)
    if (restoreFocus) triggerRef.current?.focus()
  }, [])

  // Measure the rendered menu (it mounts hidden at 0,0) and place it below the
  // trigger, right-aligned. Flip above only when it does not fit below and there
  // is more room above.
  useLayoutEffect(() => {
    if (!open || !triggerRef.current || !menuRef.current) return
    const r = triggerRef.current.getBoundingClientRect()
    const menuHeight = menuRef.current.getBoundingClientRect().height
    const gap = 4
    const spaceBelow = window.innerHeight - r.bottom - gap
    const spaceAbove = r.top - gap
    const flipUp = menuHeight > spaceBelow && spaceAbove > spaceBelow
    const top = flipUp ? Math.max(gap, r.top - gap - menuHeight) : r.bottom + gap
    const left = Math.max(gap, Math.min(r.right - MENU_WIDTH, window.innerWidth - MENU_WIDTH - gap))
    setPos({ top, left })
  }, [open, actions.length])

  useEffect(() => {
    if (open && pos) enabledItems()[0]?.focus()
    // Focus only once the menu has been positioned.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, pos !== null])

  useEffect(() => {
    if (!open) return
    const onPointerDown = (ev: MouseEvent) => {
      const t = ev.target as Node
      if (menuRef.current?.contains(t) || triggerRef.current?.contains(t)) return
      close(false)
    }
    const dismiss = () => close(false)
    document.addEventListener('mousedown', onPointerDown)
    window.addEventListener('resize', dismiss)
    window.addEventListener('scroll', dismiss, true)
    return () => {
      document.removeEventListener('mousedown', onPointerDown)
      window.removeEventListener('resize', dismiss)
      window.removeEventListener('scroll', dismiss, true)
    }
  }, [open, close])

  const onMenuKeyDown = (ev: React.KeyboardEvent) => {
    const items = enabledItems()
    const idx = items.indexOf(document.activeElement as HTMLButtonElement)
    switch (ev.key) {
      case 'Escape':
        ev.preventDefault()
        close()
        break
      case 'Tab':
        close(false)
        break
      case 'ArrowDown':
        ev.preventDefault()
        items[(idx + 1) % items.length]?.focus()
        break
      case 'ArrowUp':
        ev.preventDefault()
        items[(idx - 1 + items.length) % items.length]?.focus()
        break
      case 'Home':
        ev.preventDefault()
        items[0]?.focus()
        break
      case 'End':
        ev.preventDefault()
        items[items.length - 1]?.focus()
        break
    }
  }

  const toggle = () => {
    if (open) close()
    else {
      setPos(null)
      setOpen(true)
    }
  }

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        aria-label={`Actions for ${label}`}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={toggle}
        onKeyDown={(ev) => {
          if (ev.key === 'ArrowDown' && !open) {
            ev.preventDefault()
            toggle()
          }
        }}
        className="inline-flex h-11 w-11 items-center justify-center rounded-sm text-slate-600 hover:bg-slate-100 hover:text-slate-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-8 md:w-8"
      >
        <svg aria-hidden="true" viewBox="0 0 16 16" className="h-4 w-4" fill="currentColor">
          <circle cx="8" cy="3" r="1.5" />
          <circle cx="8" cy="8" r="1.5" />
          <circle cx="8" cy="13" r="1.5" />
        </svg>
      </button>
      {open &&
        createPortal(
          <div
            ref={menuRef}
            id={menuId}
            role="menu"
            aria-label={`Actions for ${label}`}
            onKeyDown={onMenuKeyDown}
            style={{ position: 'fixed', top: pos?.top ?? 0, left: pos?.left ?? 0, width: MENU_WIDTH, visibility: pos ? 'visible' : 'hidden' }}
            className="z-50 rounded-sm border border-slate-200 bg-white py-1 shadow-md"
          >
            {actions.map((a) => (
              <button
                key={a.label}
                type="button"
                role="menuitem"
                disabled={a.disabled}
                title={a.title}
                tabIndex={-1}
                onClick={() => {
                  close()
                  a.onSelect()
                }}
                className={`flex h-9 w-full items-center px-3 text-left text-sm focus:outline-none disabled:cursor-not-allowed disabled:text-slate-500 ${
                  a.danger ? 'text-red-600 hover:bg-red-50 focus:bg-red-50' : 'text-slate-900 hover:bg-slate-100 focus:bg-slate-100'
                }`}
              >
                {a.label}
              </button>
            ))}
          </div>,
          document.body,
        )}
    </>
  )
}
