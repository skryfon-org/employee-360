import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '../../stores/authStore'

/** Layout route: renders children only for users holding one of `roles`, else redirects to the dashboard. The API still enforces access. */
export function RequireRole({ roles }: { roles: string[] }) {
  const userRoles = useAuthStore((s) => s.user?.roles)
  if (!userRoles?.some((r) => roles.includes(r))) return <Navigate to="/" replace />
  return <Outlet />
}
