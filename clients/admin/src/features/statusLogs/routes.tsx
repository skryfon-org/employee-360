import { Route } from 'react-router-dom'
import StatusLogsPage from './pages/StatusLogsPage'

/** Authenticated routes; mount inside the RequireAuth + AdminShell layout. */
export const statusLogRoutes = <Route path="/status-logs" element={<StatusLogsPage />} />
