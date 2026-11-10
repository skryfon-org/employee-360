import { useRef } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorCode, getErrorMessage } from '@employee360/api-client'
import { FormField, SubmitButton } from '@employee360/ui'
import { DESCRIPTION_MAX, holidayFormSchema, type HolidayFormValues } from '../schemas/holidaySchemas'
import {
  useCreateHolidayMutation,
  useUpdateHolidayMutation,
  type Holiday,
  type HolidayCategory,
} from '../queries/holidayQueries'
import { useDialogFocus } from '../../../hooks/useDialogFocus'
import { useToast } from '../../../hooks/useToast'

const CONFLICT = 'CONFLICT'
const fieldCls =
  'w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 placeholder:text-slate-500 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 aria-[invalid=true]:focus:ring-red-600 sm:text-sm'

export function HolidayFormModal({
  holiday,
  categories,
  categoriesLoading = false,
  categoriesError,
  defaultDate,
  onClose,
}: {
  /** Present when editing; absent when creating. */
  holiday?: Holiday
  categories: HolidayCategory[]
  categoriesLoading?: boolean
  categoriesError?: string
  /** Pre-filled date for a new holiday (kept inside the selected year). */
  defaultDate: string
  onClose: () => void
}) {
  const toast = useToast()
  const create = useCreateHolidayMutation()
  const update = useUpdateHolidayMutation()
  const editing = Boolean(holiday)
  const pending = create.isPending || update.isPending
  const dialogRef = useRef<HTMLDivElement>(null)
  const cancelRef = useRef<HTMLButtonElement>(null)
  useDialogFocus(dialogRef, cancelRef, onClose, pending)

  const {
    register,
    handleSubmit,
    setError,
    setFocus,
    control,
    formState: { errors },
  } = useForm<HolidayFormValues>({
    resolver: zodResolver(holidayFormSchema),
    defaultValues: {
      name: holiday?.name ?? '',
      date: holiday?.date ?? defaultDate,
      category_id: holiday?.category_id ?? '',
      description: holiday?.description ?? '',
    },
  })
  const descLength = [...(useWatch({ control, name: 'description' }) ?? '')].length

  const onSubmit = (v: HolidayFormValues) => {
    const handlers = {
      onSuccess: () => {
        toast.success(editing ? `Holiday "${v.name}" updated.` : `Holiday "${v.name}" added.`)
        onClose()
      },
      onError: (err: unknown) => {
        if (getErrorCode(err) === CONFLICT) {
          setError('name', { type: 'server', message: 'A holiday with this name already exists on this date.' })
          setFocus('name')
          return
        }
        toast.error(getErrorMessage(err, editing ? 'Could not update holiday.' : 'Could not add holiday.'))
      },
    }
    if (holiday) update.mutate({ id: holiday.id, body: v }, handlers)
    else create.mutate(v, handlers)
  }

  const title = editing ? 'Edit holiday' : 'Add holiday'
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="holiday-form-title"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-md overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <h2 id="holiday-form-title" className="text-base font-semibold text-slate-900">
          {title}
        </h2>
        <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label={title} className="mt-4 flex flex-col gap-4">
          <FormField label="Name" autoComplete="off" error={errors.name?.message} {...register('name')} />
          <FormField label="Date" type="date" error={errors.date?.message} {...register('date')} />
          <div className="flex flex-col gap-1">
            <label htmlFor="category_id" className="text-xs font-medium text-slate-900">
              Category
            </label>
            <select
              id="category_id"
              aria-invalid={errors.category_id ? true : undefined}
              aria-describedby={errors.category_id ? 'category_id-error' : undefined}
              className={`h-11 md:h-9 ${fieldCls}`}
              {...register('category_id')}
            >
              <option value="">Select a category</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
            {categoriesLoading && <p className="text-xs text-slate-600">Loading categories...</p>}
            {categoriesError && (
              <p role="alert" className="text-xs text-red-700">
                {categoriesError}
              </p>
            )}
            {!categoriesLoading && !categoriesError && categories.length === 0 && (
              <p role="alert" className="text-xs text-red-700">
                No active holiday categories exist for your organisation. Create one before adding holidays.
              </p>
            )}
            {errors.category_id && (
              <p id="category_id-error" role="alert" className="text-xs text-red-700">
                {errors.category_id.message}
              </p>
            )}
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="description" className="text-xs font-medium text-slate-900">
              Description (optional)
            </label>
            <textarea
              id="description"
              rows={3}
              aria-invalid={errors.description ? true : undefined}
              aria-describedby={errors.description ? 'description-error' : 'description-hint'}
              className={`py-2 ${fieldCls}`}
              {...register('description')}
            />
            {errors.description ? (
              <p id="description-error" role="alert" className="text-xs text-red-700">
                {errors.description.message}
              </p>
            ) : (
              <p id="description-hint" className="text-xs text-slate-600">
                {descLength}/{DESCRIPTION_MAX} characters.
              </p>
            )}
          </div>
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
            <SubmitButton loading={pending}>{editing ? 'Save changes' : 'Add holiday'}</SubmitButton>
          </div>
        </form>
      </div>
    </div>
  )
}
