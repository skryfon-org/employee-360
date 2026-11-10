import type { MyProfile } from '../queries/profileQueries'
import { humanize } from '../schemas/profileSchemas'
import { Section } from './fields'

/** Employment details are set by an admin; the employee sees them read-only. */
export function EmploymentReadOnly({ data }: { data?: MyProfile }) {
  const profile = data?.profile
  const rows: [string, string | undefined][] = [
    ['Employee code', profile?.employee_code],
    ['Department', data?.department_name],
    ['Position', data?.position_name],
    ['Employment type', profile?.employment_type ? humanize(profile.employment_type) : undefined],
    ['Employment status', profile?.employment_status ? humanize(profile.employment_status) : undefined],
    ['Joining date', profile?.joining_date],
    ['Probation end date', profile?.probation_end_date],
    ['Confirmation date', profile?.confirmation_date],
    ['Work location', profile?.work_location],
  ]
  return (
    <Section title="Employment" description="Managed by your administrator. Contact them if something looks wrong.">
      <dl className="grid grid-cols-1 gap-4 text-sm sm:grid-cols-2">
        {rows.map(([label, value]) => (
          <div key={label}>
            <dt className="text-xs font-medium text-slate-600">{label}</dt>
            <dd className="text-slate-900">{value || '—'}</dd>
          </div>
        ))}
      </dl>
    </Section>
  )
}
