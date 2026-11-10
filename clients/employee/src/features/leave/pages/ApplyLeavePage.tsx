import { Card, PageContainer, PageHeader } from '@employee360/ui'
import { useNavigate } from 'react-router-dom'
import { ApplyLeaveForm } from '../components/ApplyLeaveForm'

/** Create page. The backend has no update endpoint for leave requests, so editing is not supported. */
export default function ApplyLeavePage() {
  const navigate = useNavigate()
  const back = () => navigate('/leave')
  return (
    <PageContainer>
      <PageHeader title="Apply for leave" description="Submit a leave request to your approver." />
      <Card>
        <ApplyLeaveForm onSuccess={back} onCancel={back} />
      </Card>
    </PageContainer>
  )
}
