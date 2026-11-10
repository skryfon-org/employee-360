import { AcceptInvitationPage as SharedAcceptInvitationPage } from '@employee360/ui'
import { useAuthStore } from '../../../stores/authStore'

/** Thin wrapper: supplies this app's portal-specific copy, redirect URLs and session clearing. */
export default function AcceptInvitationPage() {
  return (
    <SharedAcceptInvitationPage
      currentApp="admin"
      signInLabel="Sign in to Admin"
      otherPortalName="Employee Portal"
      otherPortalNameInAcceptedHint="Employee Portal"
      employeeAppUrl={import.meta.env.VITE_EMPLOYEE_APP_URL}
      onAccepted={() => useAuthStore.getState().clear()}
    />
  )
}
