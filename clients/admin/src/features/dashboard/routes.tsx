import { Route } from 'react-router-dom'
import DashboardPage from './pages/DashboardPage'

/** Authenticated routes; mount inside the RequireAuth + AdminShell layout. */
export const dashboardRoutes = (
  <>
    <Route path="/" element={<DashboardPage />} />
    <Route path="/dashboard" element={<DashboardPage />} />
  </>
)
