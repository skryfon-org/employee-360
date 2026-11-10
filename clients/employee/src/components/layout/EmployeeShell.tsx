import { Outlet } from 'react-router-dom'
import { BrandMark, LiveClock, ThemeToggle } from '@employee360/ui'
import { useUiStore } from '../../stores/uiStore'
import { Sidebar } from './Sidebar'
import { EmployeeBreadcrumbs } from './EmployeeBreadcrumbs'

const FOCUS = 'focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'

export function EmployeeShell() {
  const setMobileOpen = useUiStore((s) => s.setSidebarMobileOpen)
  const mobileOpen = useUiStore((s) => s.sidebarMobileOpen)
  return (
    <div className="flex h-dvh overflow-hidden bg-slate-50">
      <Sidebar />
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <header className="z-20 flex min-h-14 shrink-0 flex-wrap items-center gap-x-2 border-b border-slate-200 bg-white px-3 sm:gap-x-4 sm:px-4 md:flex-nowrap">
          <div className="flex h-14 items-center gap-2 sm:gap-4 md:hidden">
            <button
              type="button"
              onClick={() => setMobileOpen(true)}
              aria-label="Open menu"
              aria-expanded={mobileOpen}
              className={`inline-flex h-11 w-11 items-center justify-center rounded-sm border border-slate-300 bg-white text-slate-800 hover:bg-slate-100 md:hidden ${FOCUS}`}
            >
              <svg aria-hidden="true" viewBox="0 0 24 24" width={20} height={20} fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round">
                <path d="M4 7h16M4 12h16M4 17h16" />
              </svg>
            </button>
            <span className="md:hidden">
              <BrandMark name="Employee360" subtitle="Employee Portal" compactOnMobile />
            </span>
          </div>
          <EmployeeBreadcrumbs />
          <LiveClock />
          <ThemeToggle />
        </header>
        <main className="relative min-h-0 min-w-0 flex-1 overflow-y-auto overscroll-contain [scrollbar-gutter:stable] px-4 py-6 sm:px-6 lg:px-8">
          <div className="mx-auto w-full max-w-6xl">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}
