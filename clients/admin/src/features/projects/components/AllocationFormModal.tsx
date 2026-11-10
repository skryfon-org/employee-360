import { useEffect, useMemo, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage, type AllocationWriteResult } from '@employee360/api-client'
import { FormField, SubmitButton } from '@employee360/ui'
import { buildAllocationFormSchema, type AllocationFormValues } from '../schemas/projectSchemas'
import {
  useCreateAllocationMutation,
  useUpdateAllocationMutation,
  type EmployeeSummary,
  type Project,
  type ProjectAllocation,
} from '../queries/projectQueries'
import { EmployeeCombobox } from './EmployeeCombobox'
import { ProjectCombobox } from './ProjectCombobox'
import { fullName } from '../../employees/employeeName'
import { useDialogFocus } from '../../../hooks/useDialogFocus'
import { useToast } from '../../../hooks/useToast'

interface Props {
  /** Required unless `userId` is set (employee-fixed mode), where the project is picked in the form. */
  projectId?: string
  /** Employee-fixed mode: the employee is given and the admin picks an active project. */
  userId?: string
  /** The project's window; allocation dates are validated against it. */
  projectStartDate?: string
  projectEndDate?: string | null
  /** Present when editing; the employee cannot be changed. */
  allocation?: ProjectAllocation
  employeeName?: string
  /** Called after a successful create/update with the saved start and end dates (end is null when open-ended). */
  onSaved?: (startDate: string, endDate: string | null) => void
  onClose: () => void
}

export function AllocationFormModal({ projectId, userId, projectStartDate, projectEndDate, allocation, employeeName, onSaved, onClose }: Props) {
  const employeeFixed = Boolean(userId) && !allocation
  const [project, setProject] = useState<Project | null>(null)
  const winStart = employeeFixed ? project?.start_date : projectStartDate
  const winEnd = employeeFixed ? project?.end_date : projectEndDate
  const schema = useMemo(
    () => buildAllocationFormSchema({ start_date: winStart, end_date: winEnd }),
    [winStart, winEnd],
  )
  const toast = useToast()
  const create = useCreateAllocationMutation()
  const update = useUpdateAllocationMutation()
  const editing = Boolean(allocation)
  const pending = create.isPending || update.isPending
  const dialogRef = useRef<HTMLDivElement>(null)
  const cancelRef = useRef<HTMLButtonElement>(null)
  useDialogFocus(dialogRef, cancelRef, onClose, pending)

  const [chosen, setChosen] = useState<EmployeeSummary | null>(null)

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    trigger,
    getValues,
    formState: { errors, isSubmitted },
  } = useForm<AllocationFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      user_id: allocation?.user_id ?? userId ?? '',
      allocation_percent: allocation?.allocation_percent !== undefined ? String(allocation.allocation_percent) : '',
      role_on_project: allocation?.role_on_project ?? '',
      start_date: allocation?.start_date ?? '',
      end_date: allocation?.end_date ?? '',
    },
  })
  register('user_id')
  // A changed project window changes what is valid: re-check dates already entered (trigger sets or
  // clears each error). Skipped on mount so a pristine prefilled form shows no errors.
  const windowKey = useRef(`${winStart}|${winEnd}`)
  useEffect(() => {
    const key = `${winStart}|${winEnd}`
    if (windowKey.current === key) return
    windowKey.current = key
    const filled = (['start_date', 'end_date'] as const).filter((f) => getValues(f))
    if (filled.length) void trigger(filled)
  }, [winStart, winEnd, getValues, trigger])
  const [projectError, setProjectError] = useState<string | undefined>()
  const pct = Number(watch('allocation_percent'))
  const resultingTotal =
    chosen?.allocated_percent !== undefined && Number.isFinite(pct) && pct > 0 ? chosen.allocated_percent + pct : null

  const onSubmit = (v: AllocationFormValues) => {
    const common = {
      allocation_percent: Number(v.allocation_percent),
      role_on_project: v.role_on_project || undefined,
      start_date: v.start_date,
      // Optional: null = open-ended; on edit, null clears a previously set end date.
      // The generated request type models the field as `string | undefined`, so widen it for the null.
      end_date: (v.end_date || null) as string | undefined,
    }
    const handlers = {
      onSuccess: (res: AllocationWriteResult) => {
        toast.success(editing ? 'Allocation updated.' : 'Allocation added.')
        // Warn-only: the allocation is already saved.
        if (res.over_allocated) toast.warning(`This employee is now allocated ${res.user_total_percent}% in total (over 100%).`)
        onSaved?.(v.start_date, v.end_date || null)
        onClose()
      },
      onError: (err: unknown) => toast.error(getErrorMessage(err, editing ? 'Could not update allocation.' : 'Could not add allocation.')),
    }
    if (allocation?.id) update.mutate({ id: allocation.id, body: common }, handlers)
    else if (employeeFixed) {
      if (!project?.id) return setProjectError('Select a project')
      create.mutate({ projectId: project.id, body: { user_id: v.user_id, ...common } }, handlers)
    } else create.mutate({ projectId: projectId as string, body: { user_id: v.user_id, ...common } }, handlers)
  }

  const title = editing ? 'Edit allocation' : 'Add allocation'
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="allocation-form-title"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-lg overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <h2 id="allocation-form-title" className="text-base font-semibold text-slate-900">
          {title}
        </h2>
        <form onSubmit={handleSubmit(onSubmit, () => {
          if (employeeFixed && !project?.id) setProjectError('Select a project')
        })} noValidate aria-label={title} className="mt-4 flex flex-col gap-4">
          {editing ? (
            <p className="text-sm text-slate-900">
              <span className="text-xs font-medium text-slate-600">Employee</span>
              <br />
              {employeeName ?? allocation?.user_id}
            </p>
          ) : employeeFixed ? (
            <ProjectCombobox
              label="Project"
              required
              selected={project}
              error={projectError}
              onSelect={(p) => {
                setProject(p)
                setProjectError(p ? undefined : isSubmitted ? 'Select a project' : undefined)
              }}
            />
          ) : (
            <>
              <EmployeeCombobox
                label="Employee"
                required
                selected={chosen}
                error={errors.user_id?.message}
                onSelect={(emp) => {
                  setChosen(emp)
                  setValue('user_id', emp?.id ?? '', { shouldValidate: Boolean(emp) || isSubmitted })
                }}
              />
            </>
          )}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField
              label="Allocation %"
              required
              inputMode="numeric"
              autoComplete="off"
              error={errors.allocation_percent?.message}
              {...register('allocation_percent')}
            />
            <FormField label="Role on project" autoComplete="off" error={errors.role_on_project?.message} {...register('role_on_project')} />
            <FormField label="Start date" type="date" required min={winStart} max={winEnd ?? undefined} error={errors.start_date?.message} {...register('start_date')} />
            <FormField label="End date (optional)" type="date" error={errors.end_date?.message} {...register('end_date')} />
          </div>
          {winStart && (
            <p className="text-xs text-slate-600">
              Project runs {winStart} to {winEnd ?? 'ongoing'}; allocations must fall within it.
            </p>
          )}
          {resultingTotal !== null && (
            <p role="status" className={`text-xs ${resultingTotal > 100 ? 'font-medium text-amber-800' : 'text-slate-600'}`}>
              {resultingTotal > 100
                ? `Warning: this would bring ${chosen ? fullName(chosen) : 'the employee'} to ${resultingTotal}% allocated in total (over 100%). You can still save.`
                : `Resulting total: ${resultingTotal}% allocated.`}
            </p>
          )}
          <p className="text-xs text-slate-600">Totals above 100% are allowed and only shown as a warning.</p>
          <div className="mt-2 flex justify-end gap-2">
            <button
              ref={cancelRef}
              type="button"
              onClick={onClose}
              disabled={pending}
              className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500 md:h-9"
            >
              Cancel
            </button>
            <SubmitButton loading={pending}>{editing ? 'Save changes' : 'Add allocation'}</SubmitButton>
          </div>
        </form>
      </div>
    </div>
  )
}
