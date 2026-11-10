import { create } from 'zustand'
import { useAuthStore } from './authStore'

interface ProfileGateState {
  /** Set when any API call returned 403 `profile_incomplete`; forces the gate until /me confirms otherwise. */
  profileIncomplete: boolean
  /** The welcome screen was acknowledged this session (persisted in sessionStorage so a reload keeps it; cleared on sign-out). */
  welcomeDismissed: boolean
  setProfileIncomplete: (v: boolean) => void
  dismissWelcome: () => void
  reset: () => void
}

const WELCOME_KEY = 'e360.welcomeDismissed'

function readWelcomeDismissed(): boolean {
  try {
    return sessionStorage.getItem(WELCOME_KEY) === '1'
  } catch {
    return false
  }
}

function writeWelcomeDismissed(v: boolean): void {
  try {
    if (v) sessionStorage.setItem(WELCOME_KEY, '1')
    else sessionStorage.removeItem(WELCOME_KEY)
  } catch {
    /* storage unavailable: fall back to in-memory only */
  }
}

/** Ephemeral gate state only. Profile data itself lives in TanStack Query. */
export const useProfileGateStore = create<ProfileGateState>()((set) => ({
  profileIncomplete: false,
  welcomeDismissed: readWelcomeDismissed(),
  setProfileIncomplete: (profileIncomplete) => set({ profileIncomplete }),
  dismissWelcome: () => {
    writeWelcomeDismissed(true)
    set({ welcomeDismissed: true })
  },
  reset: () => {
    writeWelcomeDismissed(false)
    set({ profileIncomplete: false, welcomeDismissed: false })
  },
}))

// A new session (sign out, expired refresh) starts with a clean gate.
useAuthStore.subscribe((s, prev) => {
  if (prev.user && !s.user) useProfileGateStore.getState().reset()
})
