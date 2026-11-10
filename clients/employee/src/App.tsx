import { Navigate, Route, Routes } from 'react-router-dom'
import { authRoutes } from './features/auth/routes'
import { RequireAuth } from './components/layout/RequireAuth'
import { EmployeeShell } from './components/layout/EmployeeShell'
import { holidayRoutes } from './features/holidays/routes'
import { projectRoutes } from './features/projects/routes'
import { ProfileGate } from './components/layout/ProfileGate'
import { profileRoutes, profileSetupRoutes } from './features/profile/routes'
import { leaveRoutes } from './features/leave/routes'
import { statusLogRoutes } from './features/statusLog/routes'
import { dashboardRoutes } from './features/dashboard/routes'
import { AppToaster } from '@employee360/ui'

export default function App() {
  return (
    <>
      <Routes>
        {authRoutes}
        <Route element={<RequireAuth />}>
          <Route element={<ProfileGate />}>
            {profileSetupRoutes}
            <Route element={<EmployeeShell />}>
              {dashboardRoutes}
              {holidayRoutes}
              {projectRoutes}
              {statusLogRoutes}
              {leaveRoutes}
              {profileRoutes}
            </Route>
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      <AppToaster />
    </>
  )
}
