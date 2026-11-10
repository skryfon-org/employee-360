import { Route } from 'react-router-dom'
import MyProjectsPage from './pages/MyProjectsPage'

/** Authenticated routes; mount inside the RequireAuth + EmployeeShell layout. */
export const projectRoutes = <Route path="/projects" element={<MyProjectsPage />} />
