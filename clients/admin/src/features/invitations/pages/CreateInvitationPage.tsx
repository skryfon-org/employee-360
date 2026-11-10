import { useNavigate } from 'react-router-dom'
import { PageContainer, PageHeader } from '@employee360/ui'
import { InviteUserForm } from '../components/InviteUserForm'

export default function CreateInvitationPage() {
  const navigate = useNavigate()
  return (
    <PageContainer>
      <PageHeader
        title="Create invitation"
        description="Invite a new user to your organization."
      />
      <div className="w-full max-w-3xl">
        <InviteUserForm onSuccess={() => navigate('/invitations')} onCancel={() => navigate('/invitations')} />
      </div>
    </PageContainer>
  )
}
