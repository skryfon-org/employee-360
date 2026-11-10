import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { isAuthenticated, useAuthStore } from '../../stores/authStore'

export function RequireAuth() {
  const authed = useAuthStore(isAuthenticated)
  const location = useLocation()
  if (!authed) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  return <Outlet />
}
