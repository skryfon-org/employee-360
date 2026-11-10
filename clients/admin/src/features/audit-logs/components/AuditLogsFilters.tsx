import { useEffect } from 'react'
import { FilterBar } from '@employee360/ui'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import {
  ACTION_GROUPS,
  ENTITY_TYPES,
  dateRangeSchema,
  type AuditLogListParams,
  type DateRangeValues,
} from '../schemas/auditLogSchemas'

const SELECT =
  'h-11 rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9'
const INPUT = `${SELECT} aria-[invalid=true]:border-red-600`
const LABEL = 'text-xs font-medium text-slate-900'
const secondaryBtn =
  'h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9'
const primaryBtn =
  'h-11 rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9'

interface Props {
  params: AuditLogListParams
  hasFilters: boolean
  onChange: (patch: Partial<AuditLogListParams>) => void
  onClear: () => void
}

export function AuditLogsFilters({ params, hasFilters, onChange, onClear }: Props) {
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<DateRangeValues>({
    resolver: zodResolver(dateRangeSchema),
    defaultValues: { from: params.from ?? '', to: params.to ?? '' },
  })

  // Keep the inputs in sync when filters change from elsewhere (e.g. Clear).
  useEffect(() => {
    reset({ from: params.from ?? '', to: params.to ?? '' })
  }, [params.from, params.to, reset])

  const submit = handleSubmit((v) => onChange({ from: v.from || undefined, to: v.to || undefined }))

  return (
    <FilterBar>
      <form onSubmit={submit} noValidate aria-label="Audit log filters" className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-start">
        <div className="flex flex-col gap-1 sm:w-48">
          <label htmlFor="audit-entity" className={LABEL}>Entity type</label>
          <select
            id="audit-entity"
            className={SELECT}
            value={params.entity_type ?? ''}
            onChange={(e) => onChange({ entity_type: e.target.value || undefined })}
          >
            <option value="">All</option>
            {ENTITY_TYPES.map((t) => (
              <option key={t.value} value={t.value}>{t.label}</option>
            ))}
          </select>
        </div>
        <div className="flex flex-col gap-1 sm:w-52">
          <label htmlFor="audit-action" className={LABEL}>Action</label>
          <select
            id="audit-action"
            className={SELECT}
            value={params.action ?? ''}
            onChange={(e) => onChange({ action: e.target.value || undefined })}
          >
            <option value="">All</option>
            {ACTION_GROUPS.map((a) => (
              <option key={a.value} value={a.value}>{a.label}</option>
            ))}
          </select>
        </div>
        <div className="flex flex-col gap-1 sm:w-40">
          <label htmlFor="audit-from" className={LABEL}>From</label>
          <input
            id="audit-from"
            type="date"
            className={INPUT}
            aria-invalid={errors.from ? true : undefined}
            aria-describedby={errors.from ? 'audit-from-err' : undefined}
            {...register('from')}
          />
          {errors.from && <p id="audit-from-err" role="alert" className="text-xs text-red-700">{errors.from.message}</p>}
        </div>
        <div className="flex flex-col gap-1 sm:w-40">
          <label htmlFor="audit-to" className={LABEL}>To</label>
          <input
            id="audit-to"
            type="date"
            className={INPUT}
            aria-invalid={errors.to ? true : undefined}
            aria-describedby={errors.to ? 'audit-to-err' : undefined}
            {...register('to')}
          />
          {errors.to && <p id="audit-to-err" role="alert" className="text-xs text-red-700 sm:w-56">{errors.to.message}</p>}
        </div>
        <div className="flex gap-2 sm:items-end sm:self-start sm:pt-[1.125rem]">
          <button type="submit" className={primaryBtn}>Apply dates</button>
          {hasFilters && (
            <button type="button" className={secondaryBtn} onClick={onClear}>Clear filters</button>
          )}
        </div>
      </form>
    </FilterBar>
  )
}
