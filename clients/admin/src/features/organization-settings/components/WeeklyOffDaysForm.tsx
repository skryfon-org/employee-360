import { useEffect } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage } from '@employee360/api-client'
import { SubmitButton } from '@employee360/ui'
import { useToast } from '../../../hooks/useToast'
import { useUpdateOrgSettingsMutation, type OrgSettings } from '../queries/orgSettingsQueries'
import { WEEKDAYS, weeklyOffDaysSchema, type WeeklyOffDaysValues } from '../schemas/orgSettingsSchemas'

const DEFAULT_OFF_DAYS = [0, 6]
const toValues = (s: OrgSettings): WeeklyOffDaysValues => ({ weekly_off_days: [...(s.weekly_off_days ?? DEFAULT_OFF_DAYS)].sort((a, b) => a - b) })

export function WeeklyOffDaysForm({ settings }: { settings: OrgSettings }) {
  const toast = useToast()
  const update = useUpdateOrgSettingsMutation()
  const {
    control,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<WeeklyOffDaysValues>({ resolver: zodResolver(weeklyOffDaysSchema), defaultValues: toValues(settings) })

  useEffect(() => {
    reset(toValues(settings))
  }, [settings, reset])

  const onSubmit = (v: WeeklyOffDaysValues) => {
    update.mutate(
      {
        // The endpoint is a full update: send the stored code format unchanged.
        employee_code_prefix: settings.employee_code_prefix ?? '',
        employee_code_padding: settings.employee_code_padding ?? 4,
        employee_code_start_number: settings.employee_code_start_number ?? 1,
        weekly_off_days: v.weekly_off_days,
      },
      {
        onSuccess: () => toast.success('Weekly off days saved.'),
        onError: (err) => {
          const message = getErrorMessage(err, 'Could not save the weekly off days.')
          setError('weekly_off_days', { type: 'server', message })
        },
      },
    )
  }

  const error = errors.weekly_off_days?.message ?? errors.weekly_off_days?.root?.message

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label="Weekly off days" className="flex flex-col gap-6">
      <Controller
        control={control}
        name="weekly_off_days"
        render={({ field }) => (
          <fieldset aria-describedby={error ? 'off-days-error' : 'off-days-hint'} className="flex flex-col gap-2">
            <legend className="text-xs font-medium text-slate-900">Off days</legend>
            <div className="flex flex-wrap gap-2">
              {WEEKDAYS.map((d) => {
                const on = field.value.includes(d.value)
                return (
                  <button
                    key={d.value}
                    type="button"
                    aria-pressed={on}
                    aria-label={d.long}
                    onClick={() => field.onChange(on ? field.value.filter((x) => x !== d.value) : [...field.value, d.value].sort((a, b) => a - b))}
                    className={`h-9 min-w-14 rounded-sm border px-3 text-sm font-medium focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 ${
                      on ? 'border-slate-900 bg-slate-900 text-white hover:bg-slate-800' : 'border-slate-300 bg-white text-slate-800 hover:bg-slate-100'
                    }`}
                  >
                    {d.short}
                  </button>
                )
              })}
            </div>
            {error ? (
              <p id="off-days-error" role="alert" className="text-xs text-red-700">{error}</p>
            ) : (
              <p id="off-days-hint" className="text-xs text-slate-600">Select between one and six days.</p>
            )}
          </fieldset>
        )}
      />
      <div className="flex justify-end">
        <SubmitButton loading={update.isPending}>Save changes</SubmitButton>
      </div>
    </form>
  )
}
