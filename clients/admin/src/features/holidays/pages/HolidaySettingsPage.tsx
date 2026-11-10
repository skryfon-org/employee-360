import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage } from '@employee360/api-client'
import { Card, FormField, InlineAlert, PageContainer, PageHeader, SubmitButton } from '@employee360/ui'
import { MAX_YEAR, MIN_YEAR, settingsFormSchema, WEEK_DAYS, type SettingsFormValues } from '../schemas/settingsSchemas'
import { useHolidayCalendarSettingsQuery, useUpdateHolidaySettingsMutation } from '../queries/holidayQueries'
import { useToast } from '../../../hooks/useToast'

const selectCls =
  'h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9'
const secondaryBtn =
  'inline-flex h-11 items-center rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9'

export default function HolidaySettingsPage() {
  const toast = useToast()
  const query = useHolidayCalendarSettingsQuery()
  const update = useUpdateHolidaySettingsMutation()
  const settings = query.data

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<SettingsFormValues>({
    resolver: zodResolver(settingsFormSchema),
    defaultValues: { default_view: 'calendar', default_year: '', week_start_day: '0' },
  })

  useEffect(() => {
    if (!settings) return
    reset({
      default_view: settings.default_view === 'list' ? 'list' : 'calendar',
      default_year: settings.default_year ? String(settings.default_year) : '',
      week_start_day: String(settings.week_start_day ?? 0) as SettingsFormValues['week_start_day'],
    })
  }, [settings, reset])

  const onSubmit = (v: SettingsFormValues) =>
    update.mutate(
      {
        default_view: v.default_view,
        week_start_day: Number(v.week_start_day),
        // Omitting default_year clears it ("current year").
        ...(v.default_year ? { default_year: Number(v.default_year) } : {}),
      },
      {
        onSuccess: () => toast.success('Calendar settings saved.'),
        onError: (err) => toast.error(getErrorMessage(err, 'Could not save calendar settings.')),
      },
    )

  return (
    <PageContainer>
      <PageHeader
        title="Calendar settings"
        description="Defaults applied when the holiday calendar opens."
        actions={
          <Link to="/holidays" className={secondaryBtn}>
            Back to holidays
          </Link>
        }
      />
      <Card role="region" aria-label="Calendar settings">
        {query.isPending && <p className="text-sm text-slate-600">Loading settings...</p>}
        {query.isError && (
          <div className="flex flex-col items-start gap-3">
            <InlineAlert tone="error">{getErrorMessage(query.error, 'Could not load calendar settings.')}</InlineAlert>
            <button type="button" className={secondaryBtn} onClick={() => void query.refetch()}>
              Retry
            </button>
          </div>
        )}
        {settings && (
          <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label="Calendar settings" className="flex max-w-md flex-col gap-4">
            <div className="flex flex-col gap-1">
              <label htmlFor="default_view" className="text-xs font-medium text-slate-900">
                Default view
              </label>
              <select id="default_view" className={selectCls} {...register('default_view')}>
                <option value="calendar">Calendar</option>
                <option value="list">List</option>
              </select>
            </div>
            <FormField
              label="Default year"
              type="text"
              inputMode="numeric"
              autoComplete="off"
              placeholder="Current year"
              hint={`Leave blank to open on the current year. ${MIN_YEAR}-${MAX_YEAR}.`}
              error={errors.default_year?.message}
              {...register('default_year')}
            />
            <div className="flex flex-col gap-1">
              <label htmlFor="week_start_day" className="text-xs font-medium text-slate-900">
                Week starts on
              </label>
              <select id="week_start_day" className={selectCls} {...register('week_start_day')}>
                {WEEK_DAYS.map((d, i) => (
                  <option key={d} value={String(i)}>
                    {d}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex justify-end">
              <SubmitButton loading={update.isPending}>Save settings</SubmitButton>
            </div>
          </form>
        )}
      </Card>
    </PageContainer>
  )
}
