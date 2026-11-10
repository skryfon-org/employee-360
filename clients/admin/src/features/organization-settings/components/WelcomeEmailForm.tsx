import { useEffect } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage } from '@employee360/api-client'
import { SubmitButton } from '@employee360/ui'
import { useToast } from '../../../hooks/useToast'
import { useUpdateOrgSettingsMutation, type OrgSettings } from '../queries/orgSettingsQueries'
import {
  MAX_WELCOME_MESSAGE_LEN,
  fieldForServerError,
  stripServerKey,
  welcomeEmailSchema,
  type WelcomeEmailFormValues,
  type WelcomeEmailSchemaOutput,
} from '../schemas/orgSettingsSchemas'

const toValues = (s: OrgSettings): WelcomeEmailFormValues => ({ welcome_message: s.welcome_message ?? '' })

export function WelcomeEmailForm({ settings }: { settings: OrgSettings }) {
  const toast = useToast()
  const update = useUpdateOrgSettingsMutation()
  const {
    register,
    handleSubmit,
    reset,
    control,
    setError,
    formState: { errors },
  } = useForm<WelcomeEmailFormValues, unknown, WelcomeEmailSchemaOutput>({
    resolver: zodResolver(welcomeEmailSchema),
    defaultValues: toValues(settings),
  })

  useEffect(() => {
    reset(toValues(settings))
  }, [settings, reset])

  const length = [...(useWatch({ control, name: 'welcome_message' }) ?? '')].length

  const onSubmit = (parsed: WelcomeEmailSchemaOutput) => {
    update.mutate(
      {
        // The endpoint is a full update: send the stored code format unchanged.
        employee_code_prefix: settings.employee_code_prefix ?? '',
        employee_code_padding: settings.employee_code_padding ?? 4,
        employee_code_start_number: settings.employee_code_start_number ?? 1,
        // Always sent: omitting keeps the stored value, so blank must be an explicit clear.
        welcome_message: parsed.welcome_message,
      },
      {
        onSuccess: () => toast.success('Welcome email saved.'),
        onError: (err) => {
          const message = getErrorMessage(err, 'Could not save the welcome email.')
          if (fieldForServerError(message) === 'welcome_message') {
            setError('welcome_message', { type: 'server', message: stripServerKey(message) })
          } else toast.error(stripServerKey(message))
        },
      },
    )
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label="Welcome email" className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <label htmlFor="welcome-message" className="text-xs font-medium text-slate-900">
          Welcome message (optional)
        </label>
        <textarea
          id="welcome-message"
          rows={5}
          aria-invalid={errors.welcome_message ? true : undefined}
          aria-describedby={errors.welcome_message ? 'welcome-message-error' : 'welcome-message-hint'}
          className="w-full rounded-sm border border-slate-300 bg-white px-3 py-2 text-base text-slate-900 placeholder:text-slate-500 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 aria-[invalid=true]:focus:ring-red-600 sm:text-sm"
          {...register('welcome_message')}
        />
        {errors.welcome_message ? (
          <p id="welcome-message-error" role="alert" className="text-xs text-red-700">
            {errors.welcome_message.message}
          </p>
        ) : (
          <p id="welcome-message-hint" className="text-xs text-slate-600">
            Plain text, included in the welcome email sent when a new user accepts their invitation. Leave blank to use the
            default message. {length}/{MAX_WELCOME_MESSAGE_LEN} characters.
          </p>
        )}
      </div>
      <div className="flex justify-end">
        <SubmitButton loading={update.isPending}>Save changes</SubmitButton>
      </div>
    </form>
  )
}
