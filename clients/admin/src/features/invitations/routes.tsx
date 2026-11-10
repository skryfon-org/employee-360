import { Route } from 'react-router-dom'
import InvitationsPage from './pages/InvitationsPage'
import CreateInvitationPage from './pages/CreateInvitationPage'

/** Authenticated routes; mount inside the RequireAuth + AdminShell layout. */
export const invitationRoutes = (
  <>
    <Route path="/invitations" element={<InvitationsPage />} />
    <Route path="/invitations/new" element={<CreateInvitationPage />} />
  </>
)
