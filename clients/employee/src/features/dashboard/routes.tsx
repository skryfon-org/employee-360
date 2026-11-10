import { Route } from 'react-router-dom'
import DashboardPage from './pages/DashboardPage'

/** Authenticated routes; mount inside the RequireAuth + EmployeeShell layout. */
export const dashboardRoutes = <Route path="/" element={<DashboardPage />} />
