import { Route } from 'react-router-dom'
import PositionsPage from './pages/PositionsPage'

/** Authenticated routes; mount inside the RequireAuth + AdminShell layout. */
export const positionRoutes = <Route path="/positions" element={<PositionsPage />} />
