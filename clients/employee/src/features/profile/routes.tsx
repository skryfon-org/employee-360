import { Navigate, Route } from 'react-router-dom'
import ProfileLayout from './pages/ProfileLayout'
import { AddressSection, EmergencyContactsSection, EmploymentSection, OverviewSection, PersonalSection } from './pages/ProfileSections'
import { PROFILE_SECTIONS } from './sections'
import WelcomePage from './pages/WelcomePage'
import ProfileCompletionPage from './pages/ProfileCompletionPage'
import { SetupLayout } from '../../components/layout/SetupLayout'

/** Welcome + mandatory completion; mount inside RequireAuth > ProfileGate (outside the main shell). */
export const profileSetupRoutes = (
  <Route element={<SetupLayout />}>
    <Route path="/welcome" element={<WelcomePage />} />
    <Route path="/complete-profile" element={<ProfileCompletionPage />} />
  </Route>
)

/** Later profile edits; mount inside the EmployeeShell layout. */
export const profileRoutes = (
  <Route path="/profile" element={<ProfileLayout />}>
    <Route index element={<Navigate to={PROFILE_SECTIONS[0].path} replace />} />
    <Route path="overview" element={<OverviewSection />} />
    <Route path="employment" element={<EmploymentSection />} />
    <Route path="personal" element={<PersonalSection />} />
    <Route path="address" element={<AddressSection />} />
    <Route path="emergency-contacts" element={<EmergencyContactsSection />} />
  </Route>
)
