import { Navigate, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { InlineAlert, PageContainer, PageHeader, Tabs } from '@employee360/ui'
import { useAuthStore } from '../../../stores/authStore'
import { OrganizationSkeleton } from '../components/OrganizationSkeleton'
import { useTenantQuery } from '../queries/tenantQueries'
import { ORGANIZATION_BASE, visibleOrganizationTabs } from '../tabs'

/** Layout for /settings/organization/*: header + URL-driven tabs; the tenant is fetched once and shared via Outlet context. */
export default function OrganizationPage() {
  const roles = useAuthStore((s) => s.user?.roles)
  const tabs = visibleOrganizationTabs(roles)
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const segment = pathname.slice(ORGANIZATION_BASE.length).split('/').filter(Boolean)[0]
  const active = tabs.find((t) => t.path === segment)
  // Tenant detail is super_admin-only on the backend; fetch it only when the open tab needs it.
  const needsTenant = Boolean(active?.requiresTenant)
  const { data: tenant, isPending, isError, error } = useTenantQuery(needsTenant)
  if (tabs.length === 0) return <Navigate to="/" replace />
  if (!active) return <Navigate to={`${ORGANIZATION_BASE}/${tabs[0].path}`} replace />

  return (
    <PageContainer>
      <PageHeader title="Organization" description="Manage your organization's details, the email domains users can be invited with, and employee codes." />
      <Tabs
        label="Organization sections"
        items={tabs.map((t) => ({ value: t.path, label: t.label }))}
        value={active.path}
        onValueChange={(v) => navigate(`${ORGANIZATION_BASE}/${v}`)}
      >
        <div className="w-full max-w-3xl">
          {needsTenant && isPending && <OrganizationSkeleton tab={active.path} />}
          {needsTenant && isError && <InlineAlert tone="error">{getErrorMessage(error, 'Could not load organization.')}</InlineAlert>}
          {(!needsTenant || tenant) && <Outlet context={tenant} />}
        </div>
      </Tabs>
    </PageContainer>
  )
}
