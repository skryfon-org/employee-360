import { Navigate, Route, Routes } from 'react-router-dom'
import { authRoutes } from './features/auth/routes'
import { RequireAuth } from './components/layout/RequireAuth'
import { AdminShell } from './components/layout/AdminShell'
import { invitationRoutes } from './features/invitations/routes'
import { AppToaster } from './components/feedback/AppToaster'
import { tenantRoutes } from './features/tenants/routes'
import { departmentRoutes } from './features/departments/routes'
import { positionRoutes } from './features/positions/routes'
import { auditLogRoutes } from './features/audit-logs/routes'
import { holidayRoutes } from './features/holidays/routes'
import { employeeRoutes } from './features/employees/routes'
import { projectRoutes } from './features/projects/routes'
import { statusLogRoutes } from './features/statusLogs/routes'
import { leaveRoutes } from './features/leave/routes'
import { dashboardRoutes } from './features/dashboard/routes'

export default function App() {
  return (
    <>
      <Routes>
        {authRoutes}
        <Route element={<RequireAuth />}>
          <Route element={<AdminShell />}>
            {dashboardRoutes}
            {invitationRoutes}
            {employeeRoutes}
            {departmentRoutes}
            {positionRoutes}
            {projectRoutes}
            {statusLogRoutes}
            {holidayRoutes}
            {leaveRoutes}
            {auditLogRoutes}
            {tenantRoutes}
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      <AppToaster />
    </>
  )
}
