import { Route } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import ResetPasswordPage from './pages/ResetPasswordPage'
import AcceptInvitationPage from './pages/AcceptInvitationPage'

/** Public (unauthenticated) auth routes. */
export const authRoutes = (
  <>
    <Route path="/login" element={<LoginPage />} />
    <Route path="/forgot-password" element={<ForgotPasswordPage />} />
    <Route path="/reset-password" element={<ResetPasswordPage />} />
    <Route path="/accept-invitation" element={<AcceptInvitationPage />} />
  </>
)
