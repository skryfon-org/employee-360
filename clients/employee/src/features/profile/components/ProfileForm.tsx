import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage } from '@employee360/api-client'
import { AddressFields, FormField, InlineAlert, SubmitButton, useAddressBinding } from '@employee360/ui'
import { useUpdateMyProfileMutation, type MyProfile, type UpdateMyProfileInput } from '../queries/profileQueries'
import {
  BLOOD_GROUPS,
  GENDERS,
  MARITAL_STATUSES,
  addressSchema,
  humanize,
  personalSchema,
  personalWithNamesSchema,
  profileSchema,
  type ProfileValues,
} from '../schemas/profileSchemas'
import { Section, SelectField, secondaryButtonCls } from './fields'

export type ProfileFormSection = 'personal' | 'address' | 'both'

export function profileDefaults(p?: MyProfile['profile'], names?: { first_name?: string; last_name?: string }): ProfileValues {
  return {
    first_name: names?.first_name ?? '',
    last_name: names?.last_name ?? '',
    phone: p?.phone ?? '',
    date_of_birth: p?.date_of_birth ?? '',
    gender: p?.gender ?? '',
    blood_group: p?.blood_group ?? '',
    marital_status: p?.marital_status ?? '',
    nationality: p?.nationality ?? '',
    personal_email: p?.personal_email ?? '',
    address_line1: p?.address_line1 ?? '',
    address_line2: p?.address_line2 ?? '',
    city: p?.city ?? '',
    state: p?.state ?? '',
    country: p?.country ?? '',
    pincode: p?.pincode ?? '',
  }
}

const PERSONAL_KEYS = ['phone', 'date_of_birth', 'gender', 'blood_group', 'marital_status', 'nationality', 'personal_email'] as const
const ADDRESS_KEYS = ['address_line1', 'address_line2', 'city', 'state', 'country', 'pincode'] as const

/**
 * Personal and/or address fields saved with PATCH /me/profile. Used by each wizard step
 * (`personal`, `address`) and by the profile page (`both`). Only the section's fields are sent.
 */
export function ProfileForm({
  profile,
  names,
  section,
  submitLabel,
  refreshMe,
  onSaved,
  onBack,
  onCancel,
}: {
  profile?: MyProfile['profile']
  /** When given, First/Last name fields are shown and saved (only if changed). Omitted in the wizard. */
  names?: { first_name?: string; last_name?: string }
  section: ProfileFormSection
  submitLabel: string
  /** Whether saving should refresh /me (and so possibly release the gate). False inside the wizard. */
  refreshMe: boolean
  onSaved: () => void
  onBack?: () => void
  /** Edit mode on the profile pages: adds Cancel and a sticky footer. Unused by the wizard. */
  onCancel?: () => void
}) {
  // Wizard steps advance on their own; only confirm saves made from the edit-mode profile pages.
  const update = useUpdateMyProfileMutation(refreshMe, onCancel ? 'Profile updated.' : undefined)
  const schema = section === 'personal' ? (names ? personalWithNamesSchema : personalSchema) : section === 'address' ? addressSchema : profileSchema
  const form = useForm<ProfileValues>({
    // The section schema is a subset of ProfileValues; fields outside the section are not validated.
    resolver: zodResolver(schema) as never,
    defaultValues: profileDefaults(profile, names),
  })
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = form
  const addressProps = useAddressBinding(form, { country: true, city: true })

  const onSubmit = (v: ProfileValues) => {
    const keys: readonly (keyof ProfileValues)[] =
      section === 'personal' ? PERSONAL_KEYS : section === 'address' ? ADDRESS_KEYS : [...PERSONAL_KEYS, ...ADDRESS_KEYS]
    const body: Record<string, string> = {}
    for (const k of keys) body[k] = v[k].trim()
    if (names && section !== 'address') {
      for (const k of ['first_name', 'last_name'] as const) {
        const next = v[k].trim()
        if (next !== (names[k] ?? '').trim()) body[k] = next
      }
    }
    update.mutate(body as UpdateMyProfileInput, { onSuccess: onSaved })
  }

  const showPersonal = section !== 'address'
  const showAddress = section !== 'personal'
  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label={section === 'address' ? 'Address' : section === 'personal' ? 'Personal details' : 'Personal details and address'} className="flex flex-col gap-6">
      {update.isError && <InlineAlert tone="error">{getErrorMessage(update.error, 'Could not save your details.')}</InlineAlert>}
      {showPersonal && (
        <Section title="Personal details" description="Fields marked required must be filled in to continue.">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {names && (
              <>
                <FormField label="First name" required autoComplete="given-name" error={errors.first_name?.message} {...register('first_name')} />
                <FormField label="Last name" required autoComplete="family-name" error={errors.last_name?.message} {...register('last_name')} />
              </>
            )}
            <FormField label="Phone" required type="tel" autoComplete="tel" error={errors.phone?.message} {...register('phone')} />
            <FormField label="Date of birth" required type="date" error={errors.date_of_birth?.message} {...register('date_of_birth')} />
            <SelectField label="Gender" required error={errors.gender?.message} {...register('gender')}>
              <option value="">Select…</option>
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
            <FormField label="Personal email" type="email" autoComplete="email" error={errors.personal_email?.message} {...register('personal_email')} />
          </div>
        </Section>
      )}
      {showAddress && (
        <Section title="Address">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField label="Address line 1" required autoComplete="address-line1" error={errors.address_line1?.message} {...register('address_line1')} />
            <FormField label="Address line 2" autoComplete="address-line2" error={errors.address_line2?.message} {...register('address_line2')} />
            <AddressFields {...addressProps} />
          </div>
        </Section>
      )}
      <div className={`flex justify-end gap-2${onCancel ? ' sticky bottom-0 border-t border-slate-200 bg-white pt-3' : ''}`}>
        {onCancel && (
          <button type="button" className={secondaryButtonCls} onClick={onCancel} disabled={update.isPending}>
            Cancel
          </button>
        )}
        {onBack && (
          <button type="button" className={secondaryButtonCls} onClick={onBack} disabled={update.isPending}>
            Back
          </button>
        )}
        <SubmitButton loading={update.isPending}>{submitLabel}</SubmitButton>
      </div>
    </form>
  )
}
