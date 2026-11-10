import { Route } from 'react-router-dom'
import AuditLogsPage from './pages/AuditLogsPage'

/** Authenticated routes; mount inside the RequireAuth + AdminShell layout. */
export const auditLogRoutes = <Route path="/audit-logs" element={<AuditLogsPage />} />
