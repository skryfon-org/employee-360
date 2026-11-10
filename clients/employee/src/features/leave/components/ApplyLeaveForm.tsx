import { zodResolver } from '@hookform/resolvers/zod'
import { LEAVE_ERROR_CODES, getErrorCode, getErrorMessage } from '@employee360/api-client'
import { FormField, InlineAlert, SubmitButton, useToast } from '@employee360/ui'
import { useId, useMemo, useRef, useState } from 'react'
import { Modal } from '../../../components/ui/Modal'
import { useForm } from 'react-hook-form'
import { buildApplySchema, countWorkingDays, formatShortDate, minStartDate, type ApplyFormValues } from '../schemas/leaveSchemas'
import { useApplyLeaveMutation, useLeaveApproversQuery, useLeaveTypesQuery, useRangeHolidays, useWeeklyOffDaysQuery } from '../queries/leaveQueries'

const SELECT =
  'h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 sm:text-sm md:h-9'

const DEFAULTS: ApplyFormValues = { leave_type_id: '', start_date: '', end_date: '', day_part: 'full', reason: '', approver_id: '' }

function SelectField({ id, label, error, children, ...rest }: { id: string; label: string; error?: string } & React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-xs font-medium text-slate-900">
        {label}
        <span className="text-red-600" aria-hidden="true">
          {' '}
          *
        </span>
      </label>
      <select id={id} aria-required="true" aria-invalid={error ? true : undefined} className={SELECT} {...rest}>
        {children}
      </select>
      {error && (
        <p role="alert" className="text-xs text-red-700">
          {error}
        </p>
      )}
    </div>
  )
}

const WEEKDAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

/** Create form. `onSuccess` runs after the request is saved (the page navigates back to the list). */
export function ApplyLeaveForm({ onSuccess, onCancel }: { onSuccess?: () => void; onCancel?: () => void }) {
  const toast = useToast()
  const types = useLeaveTypesQuery()
  const approvers = useLeaveApproversQuery()
  const apply = useApplyLeaveMutation()
  const [serverError, setServerError] = useState<string | null>(null)
  const [showDetails, setShowDetails] = useState(false)
  const detailsId = useId()
  const [pending, setPending] = useState<ApplyFormValues | null>(null)
  const offDaysQuery = useWeeklyOffDaysQuery()
  const offDays = offDaysQuery.data
  const schemaRef = useRef(buildApplySchema(new Date(), offDays))
  const min = useMemo(() => minStartDate(), [])

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setError,
    formState: { errors },
  } = useForm<ApplyFormValues>({
    // Resolved at validation time so the latest off days / holidays are used (holidays depend on watched fields).
    resolver: (values, context, options) => zodResolver(schemaRef.current)(values, context, options),
    defaultValues: DEFAULTS,
  })

  const start = watch('start_date')
  const end = watch('end_date')
  const rangeHolidays = useRangeHolidays(start, end)
  const halfAllowed = !!start && start === end
  const dayPart = watch('day_part')
  const holidayDates = useMemo(() => rangeHolidays?.filter((h) => h.isPublic).map((h) => h.date), [rangeHolidays])
  schemaRef.current = buildApplySchema(new Date(), offDays, holidayDates)
  const workingDays = offDays ? countWorkingDays(start, end, dayPart, offDays, holidayDates) : null
  // Public holidays that actually cost a working day (those on a weekly off day are moot).
  const excluded = (rangeHolidays ?? []).filter((h) => h.isPublic && !offDays?.includes(new Date(`${h.date}T00:00:00Z`).getUTCDay()))
  // Weekday names of weekly off days that actually occur within the selected range (Mon-first order).
  const offWeekdays = useMemo(() => {
    if (!offDays || !start || !end || end < start) return []
    const found = new Set<number>()
    const cur = new Date(`${start}T00:00:00Z`)
    const last = new Date(`${end}T00:00:00Z`)
    if (Number.isNaN(cur.getTime()) || Number.isNaN(last.getTime())) return []
    for (; cur <= last && found.size < offDays.length; cur.setUTCDate(cur.getUTCDate() + 1)) {
      if (offDays.includes(cur.getUTCDay())) found.add(cur.getUTCDay())
    }
    return [...found].sort((a, b) => ((a + 6) % 7) - ((b + 6) % 7)).map((d) => WEEKDAY_NAMES[d])
  }, [offDays, start, end])

  const hasExclusions = excluded.length > 0 || offWeekdays.length > 0

  // Validation passes first, then the user must confirm before anything is sent.
  const onSubmit = handleSubmit((v) => {
    setServerError(null)
    setPending(v)
  })

  const confirm = () => {
    if (!pending) return
    const v = pending
    apply.mutate(
      {
        leave_type_id: v.leave_type_id,
        start_date: v.start_date,
        end_date: v.end_date,
        day_part: v.day_part,
        reason: v.reason.trim(),
        approver_id: v.approver_id,
      },
      {
        onSuccess: () => {
          setPending(null)
          toast.success('Leave request submitted')
          reset(DEFAULTS)
          onSuccess?.()
        },
        onError: (err) => {
          setPending(null)
          const code = getErrorCode(err)
          const message = getErrorMessage(err, 'Could not submit your leave request.')
          if (code === 'BAD_REQUEST') setError('end_date', { message })
          else if (code === LEAVE_ERROR_CODES.START_TOO_OLD) setError('start_date', { message })
          else if (code === LEAVE_ERROR_CODES.INVALID_APPROVER) setError('approver_id', { message })
          else setServerError(message)
        },
      },
    )
  }

  const loadFailed = types.isError || approvers.isError
  const noApprovers = approvers.isSuccess && approvers.data.length === 0

  return (
    <form onSubmit={onSubmit} noValidate aria-label="Apply for leave" className="flex flex-col gap-4">
      {loadFailed && <InlineAlert tone="error">Could not load leave options. Please refresh and try again.</InlineAlert>}
      <div className="grid gap-4 sm:grid-cols-2">
        <SelectField id="leave-type" label="Leave type" error={errors.leave_type_id?.message} {...register('leave_type_id')}>
          <option value="">Select…</option>
          {(types.data ?? []).map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </SelectField>
        <SelectField id="leave-approver" label="Approver" error={errors.approver_id?.message} {...register('approver_id')}>
          <option value="">Select…</option>
          {(approvers.data ?? []).map((a) => (
            <option key={a.user_id} value={a.user_id}>
              {a.name} ({a.kind === 'manager' ? 'Manager' : 'Admin'})
            </option>
          ))}
        </SelectField>
      </div>
      {noApprovers && <p className="text-xs text-slate-600">No approvers are available for you. Contact your administrator.</p>}
      <div className="grid gap-4 sm:grid-cols-3">
        <FormField id="leave-start" type="date" label="Start date" required min={min} error={errors.start_date?.message} {...register('start_date')} />
        <FormField id="leave-end" type="date" label="End date" required min={start || min} error={errors.end_date?.message} {...register('end_date')} />
        <SelectField id="leave-day-part" label="Duration" error={errors.day_part?.message} {...register('day_part')}>
          <option value="full">Full day</option>
          <option value="first_half" disabled={!halfAllowed}>
            Half day - first half
          </option>
          <option value="second_half" disabled={!halfAllowed}>
            Half day - second half
          </option>
        </SelectField>
      </div>
      <p className="-mt-2 text-xs text-slate-600">Half days are only available when the start and end date are the same.</p>
      {workingDays !== null && (
        <div
          className="rounded-sm border border-slate-200 bg-slate-50 p-3"
          data-testid="working-days-summary"
          onKeyDown={(e) => {
            if (e.key === 'Escape' && showDetails) setShowDetails(false)
          }}
        >
          <div className="flex items-center justify-between gap-3">
            <p className="flex items-baseline gap-2 text-sm text-slate-600" aria-live="polite">
              <span>Working days:</span>
              <span className="text-2xl font-semibold leading-none text-slate-900">{workingDays}</span>
            </p>
            {hasExclusions && (
              <button
                type="button"
                aria-label={showDetails ? 'Hide details' : 'Show details'}
                aria-expanded={showDetails}
                aria-controls={detailsId}
                onClick={() => setShowDetails((v) => !v)}
                className="-my-2 -mr-2 inline-flex h-11 w-11 items-center justify-center rounded-sm text-slate-600 hover:text-slate-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 sm:h-9 sm:w-9"
              >
                <svg viewBox="0 0 20 20" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                  <circle cx="10" cy="10" r="8" />
                  <path d="M10 9v5" strokeLinecap="round" />
                  <circle cx="10" cy="6.25" r="0.5" fill="currentColor" />
                </svg>
              </button>
            )}
          </div>
          {hasExclusions && showDetails && (
            <div id={detailsId} className="mt-3 border-t border-slate-200 pt-3">
              <p className="text-xs font-medium text-slate-900">Excluded</p>
              <ul className="mt-2 flex flex-col divide-y divide-slate-200 text-sm text-slate-900">
                {offWeekdays.length > 0 && (
                  <li className="flex items-baseline justify-between gap-3 py-1.5 first:pt-0 last:pb-0">
                    <span>Weekly off ({offWeekdays.join(', ')})</span>
                  </li>
                )}
                {excluded.map((h) => (
                  <li key={`${h.date}-${h.name}`} className="flex items-baseline justify-between gap-3 py-1.5 first:pt-0 last:pb-0">
                    <span className="min-w-0 break-words">{h.name}</span>
                    <span className="shrink-0 text-slate-600">{formatShortDate(h.date)}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
      <div className="flex flex-col gap-1">
        <label htmlFor="leave-reason" className="text-xs font-medium text-slate-900">
          Reason
        </label>
        <textarea
          id="leave-reason"
          rows={3}
          aria-invalid={errors.reason ? true : undefined}
          className="w-full rounded-sm border border-slate-300 bg-white px-3 py-2 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 sm:text-sm"
          {...register('reason')}
        />
        {errors.reason && (
          <p role="alert" className="text-xs text-red-700">
            {errors.reason.message}
          </p>
        )}
      </div>
      {serverError && <InlineAlert tone="error">{serverError}</InlineAlert>}
      <div className="flex justify-end gap-2">
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            className="h-9 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
          >
            Cancel
          </button>
        )}
        <SubmitButton loading={false}>Submit request</SubmitButton>
      </div>
      {pending && (
        <Modal title="Confirm leave request" onClose={() => !apply.isPending && setPending(null)}>
          <dl className="mb-4 grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
            <dt className="text-slate-600">Leave type</dt>
            <dd className="text-slate-900">{types.data?.find((t) => t.id === pending.leave_type_id)?.name}</dd>
            <dt className="text-slate-600">Dates</dt>
            <dd className="text-slate-900">
              {pending.start_date === pending.end_date ? pending.start_date : `${pending.start_date} to ${pending.end_date}`}
            </dd>
            <dt className="text-slate-600">Duration</dt>
            <dd className="text-slate-900">
              {pending.day_part === 'full' ? 'Full day' : pending.day_part === 'first_half' ? 'Half day (first half)' : 'Half day (second half)'}
            </dd>
            <dt className="text-slate-600">Approver</dt>
            <dd className="text-slate-900">{approvers.data?.find((a) => a.user_id === pending.approver_id)?.name}</dd>
            <dt className="text-slate-600">Reason</dt>
            <dd className="text-slate-900">{pending.reason.trim() || '-'}</dd>
          </dl>
          <p className="mb-4 text-sm text-slate-900">Once submitted, this request cannot be edited. You can only cancel it.</p>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              disabled={apply.isPending}
              onClick={() => setPending(null)}
              className="h-9 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500"
            >
              Go back
            </button>
            <button
              type="button"
              disabled={apply.isPending}
              aria-busy={apply.isPending}
              onClick={confirm}
              className="inline-flex h-9 items-center justify-center rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-200 disabled:text-slate-500"
            >
              Confirm &amp; submit
            </button>
          </div>
        </Modal>
      )}
    </form>
  )
}
