import { useMutation } from '@tanstack/react-query'
import {
  clearSession,
  forgotPassword,
  login,
  logout,
  resetPassword,
  type ForgotPasswordRequest,
  type LoginRequest,
  type ResetPasswordRequest,
} from '@employee360/api-client'
import { EMPLOYEE_ROLES, useAuthStore, type AuthUser } from '../../../stores/authStore'

export class NotEmployeeError extends Error {
  constructor() {
    super('Not an employee account')
    this.name = 'NotEmployeeError'
  }
}

export function useLoginMutation() {
  return useMutation({
    mutationFn: async (credentials: LoginRequest): Promise<AuthUser> => {
      const res = await login(credentials)
      const u = res.user
      const roles = (u?.roles ?? []).map((r) => r.name).filter((n): n is string => Boolean(n))
      if (!u?.id || !roles.some((r) => EMPLOYEE_ROLES.includes(r))) {
        try {
          await logout()
        } catch {
          // best-effort server-side logout; local cleanup below always runs
        } finally {
          clearSession()
          useAuthStore.getState().clear()
        }
        throw new NotEmployeeError()
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
