import { create } from 'zustand'

interface EmployeeUiState {
  /** Last list query string (filters + page), so "Back to employees" restores the filtered view. */
  listSearch: string
  setListSearch: (search: string) => void
}

/** Ephemeral UI state only; employee data lives in TanStack Query. */
export const useEmployeeUiStore = create<EmployeeUiState>()((set) => ({
  listSearch: '',
  setListSearch: (listSearch) => set({ listSearch }),
}))
