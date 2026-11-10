import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '../../../stores/authStore'

/** UX guard only; the backend enforces per-endpoint role checks. */
export function RequireAdmin() {
  const allowed = useAuthStore((s) => Boolean(s.user?.roles.some((r) => r === 'admin' || r === 'super_admin')))
  if (!allowed) return <Navigate to="/" replace />
  return <Outlet />
}
