import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage } from '@employee360/api-client'
import { AddressFields, FormField, SubmitButton, useAddressBinding } from '@employee360/ui'
import { useToast } from '../../../hooks/useToast'
import { useUpdateEmployeeMutation, type EmployeeDetail, type UpdateEmployeeInput } from '../queries/employeeQueries'
import {
  BLOOD_GROUPS,
  GENDERS,
  MARITAL_STATUSES,
  humanize,
  addressOnlySchema,
  personalAddressSchema,
  type PersonalAddressValues,
} from '../schemas/employeeSchemas'
import { buildPatch } from '../schemas/buildPatch'
import { personalDefaults } from '../schemas/employeeDefaults'
import { SelectField, Section, secondaryButtonCls } from './fields'

/** One form per sub-page: `personal` or `address` fields of the shared profile schema; only registered (dirty) fields are patched. */
export function PersonalAddressForm({ employee, part, onDone }: { employee: EmployeeDetail; part: 'personal' | 'address'; onDone?: () => void }) {
  const isPersonal = part === 'personal'
  const toast = useToast()
  const update = useUpdateEmployeeMutation()
  const form = useForm<PersonalAddressValues>({
    resolver: zodResolver(isPersonal ? personalAddressSchema : (addressOnlySchema as unknown as typeof personalAddressSchema)),
    defaultValues: personalDefaults(employee),
  })
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, dirtyFields },
  } = form
  const addressProps = useAddressBinding(form)

  const onSubmit = (v: PersonalAddressValues) => {
    const body = buildPatch(v, dirtyFields, ['date_of_birth']) as UpdateEmployeeInput
    if (Object.keys(body).length === 0) return toast.info('No changes to save.')
    update.mutate(
      { id: employee.id as string, body },
      {
        onSuccess: () => {
          toast.success(isPersonal ? 'Personal details updated.' : 'Address updated.')
          reset(v)
          onDone?.()
        },
        onError: (err) => toast.error(getErrorMessage(err, isPersonal ? 'Could not update personal details.' : 'Could not update address.')),
      },
    )
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label={isPersonal ? 'Personal details' : 'Address'} className="flex flex-col gap-6">
      {isPersonal && (
      <Section title="Personal details" description="Admins can correct these on behalf of the employee.">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField label="First name" required autoComplete="off" error={errors.first_name?.message} {...register('first_name')} />
          <FormField label="Last name" required autoComplete="off" error={errors.last_name?.message} {...register('last_name')} />
          <FormField label="Phone" type="tel" autoComplete="off" error={errors.phone?.message} {...register('phone')} />
          <FormField label="Personal email" type="email" autoComplete="off" error={errors.personal_email?.message} {...register('personal_email')} />
          <FormField label="Date of birth" type="date" error={errors.date_of_birth?.message} {...register('date_of_birth')} />
          <SelectField label="Gender" error={errors.gender?.message} {...register('gender')}>
            <option value="">Not set</option>
            {GENDERS.map((g) => (
              <option key={g} value={g}>{humanize(g)}</option>
            ))}
          </SelectField>
          <SelectField label="Blood group" error={errors.blood_group?.message} {...register('blood_group')}>
            <option value="">Not set</option>
            {BLOOD_GROUPS.map((g) => (
              <option key={g} value={g}>{g}</option>
            ))}
          </SelectField>
          <SelectField label="Marital status" error={errors.marital_status?.message} {...register('marital_status')}>
            <option value="">Not set</option>
            {MARITAL_STATUSES.map((g) => (
              <option key={g} value={g}>{humanize(g)}</option>
            ))}
          </SelectField>
          <FormField label="Nationality" error={errors.nationality?.message} {...register('nationality')} />
        </div>
      </Section>
      )}
      {!isPersonal && (
      <Section title="Address" description="Admins can correct these on behalf of the employee.">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField label="Address line 1" error={errors.address_line1?.message} {...register('address_line1')} />
          <FormField label="Address line 2" error={errors.address_line2?.message} {...register('address_line2')} />
          <AddressFields {...addressProps} />
        </div>
      </Section>
      )}
      <div className="flex justify-end gap-2 sticky bottom-0 border-t border-slate-200 bg-white pt-3">
        {onDone && (
          <button type="button" className={secondaryButtonCls} disabled={update.isPending} onClick={() => { reset(personalDefaults(employee)); onDone() }}>
            Cancel
          </button>
        )}
        <SubmitButton loading={update.isPending}>{isPersonal ? 'Save personal details' : 'Save address'}</SubmitButton>
      </div>
    </form>
  )
}
