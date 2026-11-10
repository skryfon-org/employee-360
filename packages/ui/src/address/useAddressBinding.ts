import type { FieldErrors, FieldValues, Path, PathValue, UseFormReturn } from 'react-hook-form'
import type { AddressFieldsProps, AddressFieldsValue } from './AddressFields'

const KEYS = ['country', 'state', 'city', 'pincode'] as const

/**
 * Connects <AddressFields> to a react-hook-form form that has string fields
 * `country`, `state`, `city` and `pincode`. Edits mark fields dirty; changing the
 * country revalidates the pincode against the new country's rule.
 */
export function useAddressBinding<T extends FieldValues>(
  form: UseFormReturn<T>,
  required?: AddressFieldsProps['required'],
): Pick<AddressFieldsProps, 'value' | 'onChange' | 'errors' | 'required'> {
  const { watch, setValue, trigger, formState } = form
  const [country, state, city, pincode] = watch(KEYS as unknown as Path<T>[]) as unknown as string[]
  const errs = formState.errors as FieldErrors<Record<(typeof KEYS)[number], string>>
  const revalidate = formState.isSubmitted

  return {
    value: { country: country ?? '', state: state ?? '', city: city ?? '', pincode: pincode ?? '' },
    required,
    errors: {
      country: errs.country?.message,
      state: errs.state?.message,
      city: errs.city?.message,
      pincode: errs.pincode?.message,
    },
    onChange: (patch: Partial<AddressFieldsValue>) => {
      for (const k of KEYS) {
        const v = patch[k]
        if (v === undefined) continue
        setValue(k as Path<T>, v as PathValue<T, Path<T>>, { shouldDirty: true, shouldValidate: revalidate })
      }
      if (patch.country !== undefined && (revalidate || (pincode ?? '') !== '')) void trigger('pincode' as Path<T>)
    },
  }
}
