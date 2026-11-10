import { Route } from 'react-router-dom'
import ProjectsPage from './pages/ProjectsPage'
import ProjectFormPage from './pages/ProjectFormPage'
import ProjectDetailPage from './pages/ProjectDetailPage'

/** Authenticated routes; mount inside the RequireAuth + AdminShell layout. Nav entry is admin/super_admin only. */
export const projectRoutes = (
  <>
    <Route path="/projects" element={<ProjectsPage />} />
    <Route path="/projects/new" element={<ProjectFormPage />} />
    <Route path="/projects/:id" element={<ProjectDetailPage />} />
    <Route path="/projects/:id/edit" element={<ProjectFormPage />} />
  </>
)
