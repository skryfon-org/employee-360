import { useEffect, useMemo } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage } from '@employee360/api-client'
import { FormField, InlineAlert, SubmitButton } from '@employee360/ui'
import { useToast } from '../../../hooks/useToast'
import { useUpdateOrgSettingsMutation, type OrgSettings } from '../queries/orgSettingsQueries'
import {
  MAX_CODE_LEN,
  MAX_PADDING,
  MAX_PREFIX_LEN,
  createOrgSettingsSchema,
  fieldForServerError,
  formatEmployeeCode,
  normalizePrefix,
  stripServerKey,
  type OrgSettingsSchemaOutput,
  type OrgSettingsFormValues,
} from '../schemas/orgSettingsSchemas'

function toValues(s: OrgSettings): OrgSettingsFormValues {
  return {
    employee_code_prefix: s.employee_code_prefix ?? '',
    employee_code_padding: String(s.employee_code_padding ?? 4),
    employee_code_start_number: String(s.employee_code_start_number ?? 1),
  }
}

export function EmployeeCodeForm({ settings }: { settings: OrgSettings }) {
  const toast = useToast()
  const update = useUpdateOrgSettingsMutation()
  const issued = Boolean(settings.employee_codes_issued)
  const counter = settings.employee_code_next_number ?? 1
  // Once codes exist the start number may only be raised to the counter or above.
  const minStart = issued ? counter : 1
  const schema = useMemo(() => createOrgSettingsSchema(minStart), [minStart])

  const {
    register,
    handleSubmit,
    reset,
    control,
    setError,
    formState: { errors },
  } = useForm<OrgSettingsFormValues, unknown, OrgSettingsSchemaOutput>({ resolver: zodResolver(schema), defaultValues: toValues(settings) })

  useEffect(() => {
    reset(toValues(settings))
  }, [settings, reset])

  const [prefix, padding, start] = useWatch({
    control,
    name: ['employee_code_prefix', 'employee_code_padding', 'employee_code_start_number'],
  })
  const nextNumber = issued ? String(Math.max(counter, Number(start) || 0)) : start
  const preview = formatEmployeeCode(prefix, padding, nextNumber)
  const savedPrefix = settings.employee_code_prefix ?? ''
  const prefixChanged = prefix.trim().toUpperCase() !== savedPrefix && prefix.trim() !== ''

  const prefixField = register('employee_code_prefix')

  const onSubmit = (parsed: OrgSettingsSchemaOutput) => {
    update.mutate(
      {
        employee_code_prefix: parsed.employee_code_prefix,
        employee_code_padding: Number(parsed.employee_code_padding),
        employee_code_start_number: Number(parsed.employee_code_start_number),
        // welcome_message is omitted on purpose: the API keeps the stored value.
      },
      {
        onSuccess: () => toast.success('Employee code format saved.'),
        onError: (err) => {
          const message = getErrorMessage(err, 'Could not save the employee code format.')
          const field = fieldForServerError(message)
          if (field && field !== 'welcome_message') setError(field, { type: 'server', message: stripServerKey(message) })
          else toast.error(stripServerKey(message))
        },
      },
    )
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label="Employee code format" className="flex flex-col gap-6">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <FormField
          label="Code prefix"
          required
          maxLength={MAX_PREFIX_LEN + 5}
          autoComplete="off"
          hint="Letters and digits, up to 10 characters, not ending in a digit."
          error={errors.employee_code_prefix?.message}
          {...prefixField}
          onChange={(e) => {
            e.target.value = normalizePrefix(e.target.value)
            return prefixField.onChange(e)
          }}
        />
        <FormField
          label="Digit padding"
          required
          inputMode="numeric"
          hint={`Minimum digits, 1 to ${MAX_PADDING}.`}
          error={errors.employee_code_padding?.message}
          {...register('employee_code_padding')}
        />
        <FormField
          label="Starting number"
          required
          inputMode="numeric"
          hint={issued ? `Codes have been issued; it can only be raised to ${counter} or higher.` : 'The number given to the first employee.'}
          error={errors.employee_code_start_number?.message}
          {...register('employee_code_start_number')}
        />
      </div>

      <div className="rounded-sm border border-slate-300 bg-slate-100 p-4" aria-live="polite">
        <p className="text-xs font-medium text-slate-600">Next employee code</p>
        <p className="mt-1 font-mono text-xl font-semibold text-slate-900" data-testid="code-preview">
          {preview || '-'}
        </p>
        <p className="mt-1 text-xs text-slate-600">Codes can be at most {MAX_CODE_LEN} characters in total.</p>
      </div>

      {prefixChanged && (
        <InlineAlert tone="neutral">
          Changing the prefix applies to new employees only. Existing employee codes are never rewritten.
        </InlineAlert>
      )}

      <div className="flex justify-end">
        <SubmitButton loading={update.isPending}>
          Save changes
        </SubmitButton>
      </div>
    </form>
  )
}
