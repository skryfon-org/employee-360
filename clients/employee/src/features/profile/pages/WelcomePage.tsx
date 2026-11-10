import { useNavigate } from 'react-router-dom'
import { Card, PageHeader } from '@employee360/ui'
import { useMeQuery } from '../queries/profileQueries'
import { useProfileGateStore } from '../../../stores/profileGateStore'
import { COMPLETE_PROFILE_PATH } from '../../../components/layout/ProfileGate'
import { primaryButtonCls } from '../components/fields'

/** Shown on login while `first_login` is true; continues to mandatory profile completion. */
export default function WelcomePage() {
  const { data: me } = useMeQuery()
  const dismiss = useProfileGateStore((s) => s.dismissWelcome)
  const navigate = useNavigate()
  const org = me?.organization_name
  const name = [me?.first_name, me?.last_name].filter(Boolean).join(' ')

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title={org ? `Welcome to ${org}` : 'Welcome'} description={name ? `We are glad to have you, ${name}.` : undefined} />
      <Card>
        <h2 className="text-base font-semibold text-slate-900">Let&apos;s set up your profile</h2>
        <p className="mt-2 text-sm text-slate-600">
          Before you start using the employee portal, we need a few details from you: your contact and personal
          information, your address and at least one emergency contact. It only takes a couple of minutes.
        </p>
        <div className="mt-6 flex justify-end">
          <button
            type="button"
            className={primaryButtonCls}
            onClick={() => {
              dismiss()
              navigate(COMPLETE_PROFILE_PATH, { replace: true })
            }}
          >
            Get started
          </button>
        </div>
      </Card>
    </div>
  )
}
