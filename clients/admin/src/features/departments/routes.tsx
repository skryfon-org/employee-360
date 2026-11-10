import { Route } from 'react-router-dom'
import DepartmentsPage from './pages/DepartmentsPage'

/** Authenticated routes; mount inside the RequireAuth + AdminShell layout. */
export const departmentRoutes = <Route path="/departments" element={<DepartmentsPage />} />
