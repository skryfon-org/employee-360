import { useAuthStore } from '../../../stores/authStore'
import { resolveDashboardRole } from '../schemas/dashboardSchemas'
import AdminDashboardPage from './AdminDashboardPage'
import SuperAdminDashboardPage from './SuperAdminDashboardPage'

/** Role-aware landing: only the page for the caller's role mounts, so the other endpoint is never called. */
export default function DashboardPage() {
  const roles = useAuthStore((s) => s.user?.roles)
  const role = resolveDashboardRole(roles)
  if (role === 'super_admin') return <SuperAdminDashboardPage />
  if (role === 'admin') return <AdminDashboardPage />
  return null
}
