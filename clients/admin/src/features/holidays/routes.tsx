import { Route } from 'react-router-dom'
import HolidaysPage from './pages/HolidaysPage'
import HolidayCategoriesPage from './pages/HolidayCategoriesPage'
import HolidaySettingsPage from './pages/HolidaySettingsPage'

/** Authenticated routes; mount inside the RequireAuth + AdminShell layout. */
export const holidayRoutes = (
  <>
    <Route path="/holidays" element={<HolidaysPage />} />
    <Route path="/holidays/categories" element={<HolidayCategoriesPage />} />
    <Route path="/holidays/settings" element={<HolidaySettingsPage />} />
  </>
)
