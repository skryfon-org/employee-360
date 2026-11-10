import { create } from 'zustand'

interface UiState {
  /** Desktop: sidebar shrunk to an icon rail. */
  sidebarCollapsed: boolean
  /** Small screens: sidebar drawer open. */
  sidebarMobileOpen: boolean
  toggleSidebar: () => void
  setSidebarMobileOpen: (open: boolean) => void
}

export const useUiStore = create<UiState>()((set) => ({
  sidebarCollapsed: false,
  sidebarMobileOpen: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  setSidebarMobileOpen: (open) => set({ sidebarMobileOpen: open }),
}))
