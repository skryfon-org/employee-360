import { Route } from 'react-router-dom'
import ApplyLeavePage from './pages/ApplyLeavePage'
import LeavePage from './pages/LeavePage'

/** Authenticated routes; mount inside the RequireAuth + EmployeeShell layout. */
export const leaveRoutes = (
  <>
    <Route path="/leave" element={<LeavePage />} />
    <Route path="/leave/apply" element={<ApplyLeavePage />} />
  </>
)
