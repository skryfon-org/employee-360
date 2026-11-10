import { Navigate, Route } from 'react-router-dom'
import OrganizationPage from './pages/OrganizationPage'
import { RequireAdmin } from './components/RequireAdmin'
import { ORGANIZATION_BASE, ORGANIZATION_TABS, OrganizationTabRoute } from './tabs'

/**
 * Authenticated routes for admin + super_admin; mount inside the RequireAuth + AdminShell layout.
 * Per-tab access (admins only reach Employee codes) is enforced by OrganizationPage, which redirects
 * a tab the user may not open to their first permitted tab.
 */
export const tenantRoutes = (
  <>
    <Route element={<RequireAdmin />}>
      <Route path={ORGANIZATION_BASE} element={<OrganizationPage />}>
        {/* Index/default is resolved per role by OrganizationPage. */}
        <Route index element={null} />
        {ORGANIZATION_TABS.map((tab) => (
          <Route key={tab.key} path={tab.path} element={<OrganizationTabRoute tab={tab} />} />
        ))}
      </Route>
    </Route>
    <Route path="/settings/employee-codes" element={<Navigate to={`${ORGANIZATION_BASE}/employee-codes`} replace />} />
  </>
)
