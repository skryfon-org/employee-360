import { useMutation } from '@tanstack/react-query'
import {
  forgotPassword,
  login,
  logout,
  resetPassword,
  type ForgotPasswordRequest,
  type LoginRequest,
  type ResetPasswordRequest,
} from '@employee360/api-client'
import { ADMIN_ROLES, useAuthStore, type AuthUser } from '../../../stores/authStore'

export class NotAdminError extends Error {
  constructor() {
    super('Not an admin account')
    this.name = 'NotAdminError'
  }
}

export function useLoginMutation() {
  return useMutation({
    mutationFn: async (credentials: LoginRequest): Promise<AuthUser> => {
      const res = await login(credentials)
      const u = res.user
      const roles = (u?.roles ?? []).map((r) => r.name).filter((n): n is string => Boolean(n))
      if (!u?.id || !roles.some((r) => ADMIN_ROLES.includes(r))) {
        // Admin portal only: drop the session and surface the same generic error.
        await logout().catch(() => undefined)
        throw new NotAdminError()
      }
      const user: AuthUser = {
        id: u.id,
        email: u.email ?? credentials.email ?? '',
        firstName: u.first_name,
        lastName: u.last_name,
        roles,
      }
      useAuthStore.getState().setUser(user)
      return user
    },
  })
}

export function useLogoutMutation() {
  return useMutation({
    mutationFn: logout,
    onSettled: () => useAuthStore.getState().clear(),
  })
}

export function useForgotPasswordMutation() {
  return useMutation({ mutationFn: (body: ForgotPasswordRequest) => forgotPassword(body) })
}

export function useResetPasswordMutation() {
  return useMutation({ mutationFn: (body: ResetPasswordRequest) => resetPassword(body) })
}
