import { Route } from 'react-router-dom'
import HolidaysPage from './pages/HolidaysPage'

/** Authenticated routes; mount inside the RequireAuth + EmployeeShell layout. */
export const holidayRoutes = <Route path="/holidays" element={<HolidaysPage />} />
