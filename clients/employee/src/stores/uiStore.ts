import { create } from 'zustand'

export type HolidayViewMode = 'list' | 'calendar'
export type HolidayCalendarScope = 'year' | 'month'

interface UiState {
  /** Holidays page: list or month-grid calendar. */
  holidayView: HolidayViewMode
  setHolidayView: (view: HolidayViewMode) => void
  /** Calendar view: whole year (12 mini months) or a single month. */
  holidayScope: HolidayCalendarScope
  setHolidayScope: (scope: HolidayCalendarScope) => void
  /** Desktop: sidebar shrunk to an icon rail. */
  sidebarCollapsed: boolean
  /** Small screens: sidebar drawer open. */
  sidebarMobileOpen: boolean
  toggleSidebar: () => void
  setSidebarMobileOpen: (open: boolean) => void
}

export const useUiStore = create<UiState>()((set) => ({
  holidayView: 'calendar',
  setHolidayView: (view) => set({ holidayView: view }),
  holidayScope: 'month',
  setHolidayScope: (scope) => set({ holidayScope: scope }),
  sidebarCollapsed: false,
  sidebarMobileOpen: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  setSidebarMobileOpen: (open) => set({ sidebarMobileOpen: open }),
}))
