import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage } from '@employee360/api-client'
import { FormField, SubmitButton } from '@employee360/ui'
import { useToast } from '../../../hooks/useToast'
import {
  useDepartmentOptionsQuery,
  useManagerOptionsQuery,
  usePositionOptionsQuery,
  useUpdateEmployeeMutation,
  type EmployeeDetail,
  type UpdateEmployeeInput,
} from '../queries/employeeQueries'
import {
  EMPLOYMENT_STATUSES,
  EMPLOYMENT_TYPES,
  employmentSchema,
  humanize,
  type EmploymentValues,
} from '../schemas/employeeSchemas'
import { buildPatch } from '../schemas/buildPatch'
import { useOrgSettingsQuery } from '../../organization-settings/queries/orgSettingsQueries'
import { overrideCodeError } from '../../organization-settings/schemas/orgSettingsSchemas'
import { employmentDefaults } from '../schemas/employeeDefaults'
import { SelectField, Section, secondaryButtonCls } from './fields'

const NULLABLE = [
  'joining_date',
  'probation_end_date',
  'confirmation_date',
  'exit_date',
  'department_id',
  'position_id',
  'manager_id',
] as const

/** Employment fields: admin-only (employees see them read-only on their profile page). */
export function EmploymentForm({ employee, onDone }: { employee: EmployeeDetail; onDone?: () => void }) {
  const toast = useToast()
  const update = useUpdateEmployeeMutation()
  const departments = useDepartmentOptionsQuery().data ?? []
  const positions = usePositionOptionsQuery().data ?? []
  const managers = useManagerOptionsQuery().data ?? []
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, dirtyFields },
  } = useForm<EmploymentValues>({
    resolver: zodResolver(employmentSchema),
    defaultValues: employmentDefaults(employee),
  })
  const orgSettings = useOrgSettingsQuery().data
  const [overrideCode, setOverrideCode] = useState(false)
  const assignmentChanged = Boolean(dirtyFields.department_id || dirtyFields.position_id)

  const onSubmit = (v: EmploymentValues) => {
    if (overrideCode && dirtyFields.employee_code && orgSettings?.employee_code_prefix) {
      const msg = overrideCodeError(v.employee_code, orgSettings.employee_code_prefix)
      if (msg) return setError('employee_code', { type: 'validate', message: msg })
    }
    const dirty = { ...dirtyFields }
    delete dirty.effective_date
    delete dirty.change_reason
    const patch = buildPatch(v, dirty, NULLABLE)
    if (Object.keys(patch).length === 0) return toast.info('No changes to save.')
    if (assignmentChanged) {
      if (v.effective_date) patch.effective_date = v.effective_date
      if (v.change_reason.trim()) patch.change_reason = v.change_reason.trim()
    }
    update.mutate(
      { id: employee.id as string, body: patch as UpdateEmployeeInput },
      {
        onSuccess: () => {
          toast.success('Employment details updated.')
          reset({ ...v, effective_date: '', change_reason: '' })
          setOverrideCode(false)
          onDone?.()
        },
        onError: (err) => toast.error(getErrorMessage(err, 'Could not update employment details.')),
      },
    )
  }

  // Inactive departments/positions can't be newly assigned, but the current one must stay selectable.
  const deptOptions = departments.filter((d) => d.is_active || d.id === employee.department_id)
  const posOptions = positions.filter((p) => p.is_active || p.id === employee.position_id)

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label="Employment details" className="flex flex-col gap-6">
      <Section title="Employment" description="Only admins can change these fields.">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="flex flex-col gap-2">
            <FormField
              label="Employee code"
              readOnly={!overrideCode}
              hint={
                overrideCode
                  ? `Must be ${orgSettings?.employee_code_prefix ?? 'the organization prefix'} followed by digits.`
                  : 'Generated automatically.'
              }
              error={errors.employee_code?.message}
              {...register('employee_code')}
            />
            {!overrideCode && (
              <button
                type="button"
                className="self-start text-xs font-medium text-slate-900 hover:text-slate-700"
                onClick={() => setOverrideCode(true)}
              >
                Override code
              </button>
            )}
          </div>
          <FormField label="Work location" error={errors.work_location?.message} {...register('work_location')} />
          <SelectField label="Employment type" error={errors.employment_type?.message} {...register('employment_type')}>
            <option value="">Not set</option>
            {EMPLOYMENT_TYPES.map((t) => (
              <option key={t} value={t}>{humanize(t)}</option>
            ))}
          </SelectField>
          <SelectField label="Employment status" required error={errors.employment_status?.message} {...register('employment_status')}>
            {EMPLOYMENT_STATUSES.map((t) => (
              <option key={t} value={t}>{humanize(t)}</option>
            ))}
          </SelectField>
          <FormField label="Joining date" type="date" error={errors.joining_date?.message} {...register('joining_date')} />
          <FormField label="Probation end date" type="date" error={errors.probation_end_date?.message} {...register('probation_end_date')} />
          <FormField label="Confirmation date" type="date" error={errors.confirmation_date?.message} {...register('confirmation_date')} />
          <FormField label="Exit date" type="date" error={errors.exit_date?.message} {...register('exit_date')} />
          <SelectField label="Department" error={errors.department_id?.message} {...register('department_id')}>
            <option value="">None</option>
            {deptOptions.map((d) => (
              <option key={d.id} value={d.id}>{d.name}</option>
            ))}
          </SelectField>
          <SelectField label="Position" error={errors.position_id?.message} {...register('position_id')}>
            <option value="">None</option>
            {posOptions.map((p) => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </SelectField>
          <SelectField label="Manager" error={errors.manager_id?.message} {...register('manager_id')}>
            <option value="">None</option>
            {managers
              .filter((m) => m.id !== employee.id)
              .map((m) => (
                <option key={m.id} value={m.id}>
                  {[m.first_name, m.last_name].filter(Boolean).join(' ') || m.email}
                </option>
              ))}
          </SelectField>
        </div>
        {assignmentChanged && (
          <div className="grid grid-cols-1 gap-4 rounded-sm border border-slate-300 bg-slate-100 p-4 sm:grid-cols-2">
            <p className="text-xs text-slate-600 sm:col-span-2">
              Changing the department or position adds an entry to the assignment history.
            </p>
            <FormField label="Effective date" type="date" hint="Defaults to today." error={errors.effective_date?.message} {...register('effective_date')} />
            <FormField label="Change reason" error={errors.change_reason?.message} {...register('change_reason')} />
          </div>
        )}
      </Section>
      <div className="flex justify-end gap-2 sticky bottom-0 border-t border-slate-200 bg-white pt-3">
        {onDone && (
          <button type="button" className={secondaryButtonCls} disabled={update.isPending} onClick={() => { reset(employmentDefaults(employee)); onDone() }}>
            Cancel
          </button>
        )}
        <SubmitButton loading={update.isPending}>Save employment details</SubmitButton>
      </div>
    </form>
  )
}
