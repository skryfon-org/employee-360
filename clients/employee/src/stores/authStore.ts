import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { getSession, subscribeSession, type Session } from '@employee360/api-client'

export interface AuthUser {
  id: string
  email: string
  firstName?: string
  lastName?: string
  roles: string[]
}

interface AuthState {
  /** Mirror of the api-client session (tokens live there and are persisted by it). */
  accessToken: string | null
  user: AuthUser | null
  setUser: (user: AuthUser | null) => void
  clear: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: getSession()?.accessToken ?? null,
      user: null,
      setUser: (user) => set({ user }),
      clear: () => set({ accessToken: null, user: null }),
    }),
    {
      name: 'employee360.employee.auth',
      // Tokens are owned/persisted by api-client; only the profile is persisted here.
      partialize: (s) => ({ user: s.user }),
    },
  ),
)

// Keep the token mirror in sync with the api-client session (login, silent
// refresh, logout, cross-tab changes, refresh failure).
subscribeSession((session: Session | null) => {
  if (session) useAuthStore.setState({ accessToken: session.accessToken })
  else useAuthStore.getState().clear()
})

// admin/super_admin are intentionally allowed into the employee portal: login is
// one mechanism for every role (cycle-02).
export const EMPLOYEE_ROLES = ['employee', 'admin', 'super_admin']

export function isAuthenticated(s: Pick<AuthState, 'accessToken' | 'user'>): boolean {
  return Boolean(s.accessToken && s.user && s.user.roles.some((r) => EMPLOYEE_ROLES.includes(r)))
}
