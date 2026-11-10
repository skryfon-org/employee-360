import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { getErrorMessage } from '@employee360/api-client'
import { Card, InlineAlert, PageHeader, Skeleton, SkeletonRegion } from '@employee360/ui'
import { refetchMe, useMyProfileQuery } from '../queries/profileQueries'
import { useProfileGateStore } from '../../../stores/profileGateStore'
import { ProfileForm } from '../components/ProfileForm'
import { ContactsEditor } from '../components/ContactsEditor'
import { MissingFieldsAlert } from '../components/MissingFieldsAlert'
import { primaryButtonCls, secondaryButtonCls } from '../components/fields'

const STEPS = ['Personal details', 'Address', 'Emergency contacts'] as const

/**
 * Mandatory profile completion wizard. Steps save as they go (PATCH /me/profile, contacts API);
 * Finish refetches /me, which releases the route gate once the backend reports completion.
 */
export default function ProfileCompletionPage() {
  const { data, isPending, isError, error, refetch } = useMyProfileQuery()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const setFlag = useProfileGateStore((s) => s.setProfileIncomplete)
  const [step, setStep] = useState(0)
  const [finishing, setFinishing] = useState(false)
  const [stillMissing, setStillMissing] = useState<string[] | null>(null)

  if (isPending) {
    return (
      <SkeletonRegion label="Loading your profile…" className="flex flex-col gap-4">
        <Skeleton className="h-7 w-64" />
        <Skeleton className="h-64 w-full" />
      </SkeletonRegion>
    )
  }
  if (isError || !data) {
    return (
      <div className="flex flex-col items-start gap-3">
        <InlineAlert tone="error">{getErrorMessage(error, 'Could not load your profile.')}</InlineAlert>
        <button type="button" className={secondaryButtonCls} onClick={() => void refetch()}>
          Retry
        </button>
      </div>
    )
  }

  const contacts = data.emergency_contacts ?? []

  const finish = async () => {
    setFinishing(true)
    try {
      const fresh = await refetch()
      const missing = fresh.data?.missing_fields ?? []
      if (missing.length > 0) {
        setStillMissing(missing)
        // Send the user back to the first step that has something missing.
        const personalMissing = missing.some((f) => ['phone', 'date_of_birth', 'gender'].includes(f))
        const addressMissing = missing.some((f) => ['address_line1', 'city', 'country'].includes(f))
        setStep(personalMissing ? 0 : addressMissing ? 1 : 2)
        return
      }
      setStillMissing(null)
      await refetchMe(qc)
      setFlag(false)
      navigate('/', { replace: true })
    } finally {
      setFinishing(false)
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Complete your profile" description="This is required before you can use the rest of the portal." />
      <ol className="flex flex-wrap gap-2" aria-label="Progress">
        {STEPS.map((label, i) => (
          <li
            key={label}
            aria-current={i === step ? 'step' : undefined}
            className={`inline-flex h-8 items-center rounded-sm border px-3 text-xs font-medium ${
              i === step ? 'border-slate-900 bg-slate-900 text-white' : 'border-slate-300 bg-white text-slate-600'
            }`}
          >
            {i + 1}. {label}
          </li>
        ))}
      </ol>
      <MissingFieldsAlert fields={stillMissing ?? data.missing_fields} />
      <Card>
        {step === 0 && (
          <ProfileForm
            key="personal"
            section="personal"
            profile={data.profile}
            refreshMe={false}
            submitLabel="Save and continue"
            onSaved={() => setStep(1)}
          />
        )}
        {step === 1 && (
          <ProfileForm
            key="address"
            section="address"
            profile={data.profile}
            refreshMe={false}
            submitLabel="Save and continue"
            onBack={() => setStep(0)}
            onSaved={() => setStep(2)}
          />
        )}
        {step === 2 && (
          <div className="flex flex-col gap-6">
            <ContactsEditor contacts={contacts} refreshMe={false} />
            <div className="flex justify-end gap-2">
              <button type="button" className={secondaryButtonCls} onClick={() => setStep(1)} disabled={finishing}>
                Back
              </button>
              <button type="button" className={primaryButtonCls} onClick={() => void finish()} disabled={finishing || contacts.length === 0} aria-busy={finishing}>
                Finish
              </button>
            </div>
          </div>
        )}
      </Card>
    </div>
  )
}
