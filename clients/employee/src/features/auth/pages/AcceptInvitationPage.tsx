import { AcceptInvitationPage as SharedAcceptInvitationPage } from '@employee360/ui'
import { useAuthStore } from '../../../stores/authStore'

/** Thin wrapper: supplies this app's portal-specific copy, redirect URLs and session clearing. */
export default function AcceptInvitationPage() {
  return (
    <SharedAcceptInvitationPage
      currentApp="employee"
      signInLabel="Sign in to Employee Portal"
      otherPortalName="Admin Portal"
      otherPortalNameInAcceptedHint="Admin portal"
      adminAppUrl={import.meta.env.VITE_ADMIN_APP_URL}
      onAccepted={() => useAuthStore.getState().clear()}
    />
  )
}
