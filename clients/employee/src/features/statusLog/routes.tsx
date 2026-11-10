import { Route } from 'react-router-dom'
import DailyStatusPage from './pages/DailyStatusPage'

/** Authenticated routes; mount inside the RequireAuth + EmployeeShell layout. */
export const statusLogRoutes = <Route path="/status" element={<DailyStatusPage />} />
