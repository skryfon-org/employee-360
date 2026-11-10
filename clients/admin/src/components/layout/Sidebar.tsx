import { useEffect, useRef } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { BrandMark, UserBadge, pickPrimaryRole } from '@employee360/ui'
import { useAuthStore } from '../../stores/authStore'
import { useLogoutMutation } from '../../features/auth/queries/authMutations'
import { useUiStore } from '../../stores/uiStore'
import { visibleNavItems } from './navItems'

const FOCUS = 'focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'

export function Sidebar() {
  const collapsed = useUiStore((s) => s.sidebarCollapsed)
  const mobileOpen = useUiStore((s) => s.sidebarMobileOpen)
  const toggle = useUiStore((s) => s.toggleSidebar)
  const setMobileOpen = useUiStore((s) => s.setSidebarMobileOpen)

  const user = useAuthStore((s) => s.user)
  const navigate = useNavigate()
  const logout = useLogoutMutation()
  const closeRef = useRef<HTMLButtonElement>(null)

  // Mobile drawer: lock page scroll while open, close on Escape or when resized to desktop.
  useEffect(() => {
    if (!mobileOpen) return
    const trigger = document.activeElement instanceof HTMLElement ? document.activeElement : null
    closeRef.current?.focus()
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setMobileOpen(false)
    const mq = window.matchMedia?.('(min-width: 768px)')
    const onMq = () => mq?.matches && setMobileOpen(false)
    document.addEventListener('keydown', onKey)
    mq?.addEventListener?.('change', onMq)
    return () => {
      document.body.style.overflow = prev
      if (trigger?.isConnected) trigger.focus()
      document.removeEventListener('keydown', onKey)
      mq?.removeEventListener?.('change', onMq)
    }
  }, [mobileOpen, setMobileOpen])

  // Labels stay visible in the mobile drawer even if the desktop rail is collapsed.
  const labelClass = collapsed ? 'md:hidden' : ''

  return (
    <>
      {mobileOpen && (
        <div
          data-testid="sidebar-backdrop"
          className="fixed inset-0 z-30 bg-black/50 md:hidden"
          onClick={() => setMobileOpen(false)}
        />
      )}
      <aside
        aria-label="Sidebar"
        data-collapsed={collapsed}
        className={`fixed inset-y-0 left-0 z-40 flex w-64 max-w-[85vw] flex-col overflow-y-auto border-r border-slate-200 bg-white transition-transform md:static md:translate-x-0 md:transition-none ${
          collapsed ? 'md:w-16' : 'md:w-56'
        } ${mobileOpen ? 'translate-x-0' : '-translate-x-full max-md:invisible'}`}
      >
        <div className={`flex h-14 items-center justify-between gap-2 border-b border-slate-200 px-4 ${collapsed ? 'md:justify-center md:px-0' : ''}`}>
          <span className={collapsed ? 'md:hidden' : ''}>
            <BrandMark name="Employee360" subtitle="Admin" />
          </span>
          <button
            ref={closeRef}
            type="button"
            onClick={() => setMobileOpen(false)}
            aria-label="Close menu"
            className={`inline-flex h-11 w-11 items-center justify-center rounded-sm text-slate-900 hover:bg-slate-100 md:hidden ${FOCUS}`}
          >
            <svg aria-hidden="true" viewBox="0 0 24 24" width={20} height={20} fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round">
              <path d="M6 6l12 12M18 6 6 18" />
            </svg>
          </button>
          <button
            type="button"
            onClick={toggle}
            aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            aria-expanded={!collapsed}
            className={`hidden h-8 w-8 items-center justify-center rounded-sm text-sm text-slate-900 hover:bg-slate-100 md:flex ${FOCUS}`}
          >
            <svg aria-hidden="true" viewBox="0 0 24 24" width={18} height={18} fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <path d={collapsed ? 'm13 17 5-5-5-5M6 17l5-5-5-5' : 'm11 17-5-5 5-5M18 17l-5-5 5-5'} />
            </svg>
          </button>
        </div>
        <nav aria-label="Main" className={`flex flex-1 flex-col gap-1 ${collapsed ? 'p-3 md:p-2' : 'p-3'}`}>
          <p className={`px-3 pb-1 text-xs font-medium text-slate-600 ${labelClass}`}>Manage</p>
          {visibleNavItems(user?.roles).map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              title={item.label}
              aria-label={item.label}
              onClick={() => setMobileOpen(false)}
              className={({ isActive }) =>
                `flex h-11 items-center gap-3 md:h-10 rounded-sm border-l-2 px-3 text-sm ${collapsed ? 'md:justify-center md:px-0' : ''} ${FOCUS} ${
                  isActive
                    ? 'border-slate-900 bg-slate-200 font-semibold text-slate-900'
                    : 'border-transparent text-slate-700 hover:bg-slate-100 hover:text-slate-900'
                }`
              }
            >
              <item.icon className="shrink-0" />
              <span className={labelClass}>{item.label}</span>
            </NavLink>
          ))}
        </nav>
        <div className={`mt-auto flex flex-col gap-3 border-t border-slate-200 p-3 ${collapsed ? 'md:items-center md:p-2' : ''}`}>
          <UserBadge email={user?.email} name={[user?.firstName, user?.lastName].filter(Boolean).join(' ')} role={pickPrimaryRole(user?.roles)} variant="sidebar" collapsed={collapsed} />
          <button
            type="button"
            aria-label="Sign out"
            title="Sign out"
            onClick={() => logout.mutate(undefined, { onSettled: () => navigate('/login', { replace: true }) })}
            className={`flex h-9 w-full items-center justify-center gap-2 rounded-sm border border-slate-300 bg-white px-3 text-sm font-medium text-slate-700 hover:bg-slate-100 ${collapsed ? 'md:w-9 md:px-0' : ''} ${FOCUS}`}
          >
            <svg aria-hidden="true" viewBox="0 0 24 24" width={16} height={16} fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" className="shrink-0">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" />
            </svg>
            <span className={labelClass}>Sign out</span>
          </button>
        </div>
      </aside>
    </>
  )
}
