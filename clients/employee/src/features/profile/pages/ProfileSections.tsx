import { useState, type ReactNode } from 'react'
import { Card } from '@employee360/ui'
import { useAuthStore } from '../../../stores/authStore'
import { ContactsEditor } from '../components/ContactsEditor'
import { EmploymentReadOnly } from '../components/EmploymentReadOnly'
import { ProfileForm } from '../components/ProfileForm'
import { DefinitionList, Section, secondaryButtonCls } from '../components/fields'
import { useMyProfileData } from '../hooks/useMyProfileData'
import { humanize } from '../schemas/profileSchemas'

const FOOTER = 'sticky bottom-0 flex justify-end border-t border-slate-200 bg-white pt-3'

/**
 * Card that opens in read-only view mode. "Edit <label>" swaps in the form; the form calls `done`
 * on a successful save or Cancel. Edit-mode state is local to the section.
 */
function EditableCard({ label, view, edit }: { label: string; view: ReactNode; edit: (done: () => void) => ReactNode }) {
  const [editing, setEditing] = useState(false)
  if (editing) return <Card>{edit(() => setEditing(false))}</Card>
  return (
    <Card>
      <div className="flex flex-col gap-4">
        {view}
        <div className={FOOTER}>
          <button type="button" className={secondaryButtonCls} onClick={() => setEditing(true)}>
            Edit {label}
          </button>
        </div>
      </div>
    </Card>
  )
}

export function OverviewSection() {
  const { profile: p, first_name, last_name, department_name, position_name } = useMyProfileData()
  const user = useAuthStore((s) => s.user)
  const name = [first_name ?? user?.firstName, last_name ?? user?.lastName].filter(Boolean).join(' ')
  return (
    <Card>
      <Section title="Overview" description="A summary of your profile. Use the tabs above to view or edit details.">
        <DefinitionList
          rows={[
            ['Name', name],
            ['Email', user?.email],
            ['Employee code', p?.employee_code],
            ['Department', department_name],
            ['Position', position_name],
            ['Employment status', p?.employment_status ? humanize(p.employment_status) : undefined],
            ['Work location', p?.work_location],
            ['Phone', p?.phone],
          ]}
        />
      </Section>
    </Card>
  )
}

export function EmploymentSection() {
  const data = useMyProfileData()
  return (
    <Card>
      <EmploymentReadOnly data={data} />
    </Card>
  )
}

export function PersonalSection() {
  const { profile: p, first_name, last_name } = useMyProfileData()
  return (
    <EditableCard
      key={JSON.stringify([p ?? {}, first_name, last_name])}
      label="personal details"
      view={
        <Section title="Personal details">
          <DefinitionList
            rows={[
              ['First name', first_name],
              ['Last name', last_name],
              ['Phone', p?.phone],
              ['Date of birth', p?.date_of_birth],
              ['Gender', p?.gender ? humanize(p.gender) : undefined],
              ['Blood group', p?.blood_group],
              ['Marital status', p?.marital_status ? humanize(p.marital_status) : undefined],
              ['Nationality', p?.nationality],
              ['Personal email', p?.personal_email],
            ]}
          />
        </Section>
      }
      edit={(done) => <ProfileForm section="personal" profile={p} names={{ first_name, last_name }} refreshMe submitLabel="Save personal details" onSaved={done} onCancel={done} />}
    />
  )
}

export function AddressSection() {
  const { profile: p } = useMyProfileData()
  return (
    <EditableCard
      key={JSON.stringify(p ?? {})}
      label="address"
      view={
        <Section title="Address">
          <DefinitionList
            rows={[
              ['Address line 1', p?.address_line1],
              ['Address line 2', p?.address_line2],
              ['City', p?.city],
              ['State', p?.state],
              ['Country', p?.country],
              ['Pincode', p?.pincode],
            ]}
          />
        </Section>
      }
      edit={(done) => <ProfileForm section="address" profile={p} refreshMe submitLabel="Save address" onSaved={done} onCancel={done} />}
    />
  )
}

export function EmergencyContactsSection() {
  const { emergency_contacts } = useMyProfileData()
  const [editing, setEditing] = useState(false)
  return (
    <Card>
      <div className="flex flex-col gap-4">
        <ContactsEditor contacts={emergency_contacts ?? []} refreshMe editing={editing} />
        <div className={FOOTER}>
          <button type="button" className={secondaryButtonCls} onClick={() => setEditing((v) => !v)}>
            {editing ? 'Done editing contacts' : 'Edit emergency contacts'}
          </button>
        </div>
      </div>
    </Card>
  )
}
