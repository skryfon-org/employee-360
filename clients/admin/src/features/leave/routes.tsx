import { Route } from 'react-router-dom'
import { RequireRole } from '../../components/layout/RequireRole'
import LeaveRequestsPage from './pages/LeaveRequestsPage'
import LeaveTypesPage from './pages/LeaveTypesPage'

/** Authenticated routes; mount inside the RequireAuth + AdminShell layout. Guarded to admin / super_admin (the API enforces it too). */
export const leaveRoutes = (
  <Route element={<RequireRole roles={['admin', 'super_admin']} />}>
    <Route path="/leave" element={<LeaveRequestsPage />} />
    <Route path="/leave/types" element={<LeaveTypesPage />} />
  </Route>
)
