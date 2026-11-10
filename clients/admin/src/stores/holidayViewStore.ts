import { create } from 'zustand'

export type HolidayViewMode = 'calendar' | 'list'
export type HolidayCalendarScope = 'year' | 'month'

interface HolidayViewState {
  /**
   * View-mode state (UI-only, ephemeral): `viewMode` and `calendarScope` together describe how
   * holidays are displayed. Holiday data itself lives in TanStack Query, never here.
   */
  viewMode: HolidayViewMode
  setViewMode: (mode: HolidayViewMode) => void
  /**
   * Part of the view-mode state (UI-only); only meaningful when `viewMode` is 'calendar'.
   * Whole year (12 mini months) or a single month of the selected year.
   */
  calendarScope: HolidayCalendarScope
  setCalendarScope: (scope: HolidayCalendarScope) => void
}

export const useHolidayViewStore = create<HolidayViewState>()((set) => ({
  viewMode: 'calendar',
  setViewMode: (viewMode) => set({ viewMode }),
  calendarScope: 'month',
  setCalendarScope: (calendarScope) => set({ calendarScope }),
}))
