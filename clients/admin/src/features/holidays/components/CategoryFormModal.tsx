import { useRef } from 'react'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage } from '@employee360/api-client'
import { FormField, SubmitButton, Switch } from '@employee360/ui'
import { categoryFormSchema, HEX_COLOR, type CategoryFormValues } from '../schemas/categorySchemas'
import { DESCRIPTION_MAX } from '../schemas/holidaySchemas'
import {
  useCreateHolidayCategoryMutation,
  useUpdateHolidayCategoryMutation,
  type HolidayCategory,
} from '../queries/holidayQueries'
import { isCategoryDuplicateError } from '../categoryErrors'
import { useDialogFocus } from '../../../hooks/useDialogFocus'
import { useToast } from '../../../hooks/useToast'

const DEFAULT_COLOR = '#475569'
/** Design-system category colours (the `-700` tokens) plus the neutral slate. */
const PRESET_COLORS: Array<{ name: string; hex: string }> = [
  { name: 'Green', hex: '#15803D' },
  { name: 'Amber', hex: '#B45309' },
  { name: 'Blue', hex: '#1D4ED8' },
  { name: 'Red', hex: '#B91C1C' },
  { name: 'Slate', hex: '#475569' },
]
const fieldCls =
  'w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 placeholder:text-slate-500 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 aria-[invalid=true]:focus:ring-red-600 sm:text-sm'

export function CategoryFormModal({ category, onClose }: { category?: HolidayCategory; onClose: () => void }) {
  const toast = useToast()
  const create = useCreateHolidayCategoryMutation()
  const update = useUpdateHolidayCategoryMutation()
  const editing = Boolean(category)
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
    setValue,
    formState: { errors },
  } = useForm<CategoryFormValues>({
    resolver: zodResolver(categoryFormSchema),
    defaultValues: {
      name: category?.name ?? '',
      color: category?.color || DEFAULT_COLOR,
      description: category?.description ?? '',
      is_active: category?.is_active ?? true,
      is_public: category?.is_public ?? false,
    },
  })
  const pickColor = (hex: string) => setValue('color', hex.toUpperCase(), { shouldValidate: true, shouldDirty: true })
  const color = useWatch({ control, name: 'color' })
  const descLength = [...(useWatch({ control, name: 'description' }) ?? '')].length

  const onSubmit = (v: CategoryFormValues) => {
    const handlers = {
      onSuccess: () => {
        toast.success(editing ? `Category "${v.name}" updated.` : `Category "${v.name}" created.`)
        onClose()
      },
      onError: (err: unknown) => {
        if (isCategoryDuplicateError(err)) {
          setError('name', { type: 'server', message: 'A category with this name already exists.' })
          setFocus('name')
          return
        }
        toast.error(getErrorMessage(err, editing ? 'Could not update category.' : 'Could not create category.'))
      },
    }
    if (category) update.mutate({ id: category.id, body: v }, handlers)
    else create.mutate(v, handlers)
  }

  const title = editing ? 'Edit category' : 'Add category'
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="category-form-title"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-md overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <h2 id="category-form-title" className="text-base font-semibold text-slate-900">
          {title}
        </h2>
        <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label={title} className="mt-4 flex flex-col gap-4">
          <FormField label="Name" autoComplete="off" error={errors.name?.message} {...register('name')} />
          <div className="flex flex-col gap-2">
            <span id="category-color-label" className="text-xs font-medium text-slate-900">
              Colour
            </span>
            <div role="radiogroup" aria-labelledby="category-color-label" className="flex flex-wrap items-center gap-2">
              {PRESET_COLORS.map((p) => {
                const selected = color?.toUpperCase() === p.hex
                return (
                  <button
                    key={p.hex}
                    type="button"
                    role="radio"
                    aria-checked={selected}
                    aria-label={`${p.name} ${p.hex}`}
                    title={p.name}
                    onClick={() => pickColor(p.hex)}
                    className={`flex h-11 w-11 items-center justify-center rounded-sm border focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9 md:w-9 ${
                      selected ? 'border-slate-900 ring-2 ring-slate-900 ring-offset-2' : 'border-slate-300'
                    }`}
                    style={{ backgroundColor: p.hex }}
                  >
                    {selected && (
                      <svg aria-hidden="true" viewBox="0 0 20 20" className="h-4 w-4 text-[#ffffff]" fill="currentColor">
                        <path d="M8 13.2 4.8 10l-1.4 1.4L8 16 17 7l-1.4-1.4z" />
                      </svg>
                    )}
                  </button>
                )
              })}
              <label className="flex items-center gap-2 text-xs font-medium text-slate-900">
                <input
                  type="color"
                  aria-label="Custom colour"
                  value={HEX_COLOR.test(color ?? '') ? color.toLowerCase() : DEFAULT_COLOR}
                  onChange={(e) => pickColor(e.target.value)}
                  className="h-11 w-11 cursor-pointer rounded-sm border border-slate-300 bg-white p-0.5 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9 md:w-9"
                />
                Custom
              </label>
            </div>
            <div className="flex items-center gap-2">
              <input
                id="category-color"
                type="text"
                aria-label="Hex colour"
                autoComplete="off"
                spellCheck={false}
                maxLength={7}
                placeholder="#RRGGBB"
                aria-invalid={errors.color ? true : undefined}
                aria-describedby={errors.color ? 'category-color-error' : undefined}
                className={`h-11 md:h-9 ${fieldCls}`}
                {...register('color')}
              />
              <span
                aria-hidden="true"
                data-testid="color-preview"
                className="h-11 w-11 shrink-0 rounded-sm border border-slate-300 bg-slate-100 md:h-9 md:w-9"
                style={HEX_COLOR.test(color) ? { backgroundColor: color } : undefined}
              />
            </div>
            {errors.color && (
              <p id="category-color-error" role="alert" className="text-xs text-red-700">
                {errors.color.message}
              </p>
            )}
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="category-description" className="text-xs font-medium text-slate-900">
              Description (optional)
            </label>
            <textarea
              id="category-description"
              rows={3}
              aria-invalid={errors.description ? true : undefined}
              aria-describedby={errors.description ? 'category-description-error' : 'category-description-hint'}
              className={`py-2 ${fieldCls}`}
              {...register('description')}
            />
            {errors.description ? (
              <p id="category-description-error" role="alert" className="text-xs text-red-700">
                {errors.description.message}
              </p>
            ) : (
              <p id="category-description-hint" className="text-xs text-slate-600">
                {descLength}/{DESCRIPTION_MAX} characters.
              </p>
            )}
          </div>
          <Controller
            control={control}
            name="is_active"
            render={({ field }) => (
              <Switch
                id="category-is-active"
                ref={field.ref}
                name={field.name}
                checked={field.value}
                onChange={field.onChange}
                onBlur={field.onBlur}
                label="Active"
                description="Inactive categories cannot be chosen for new holidays."
              />
            )}
          />
          <label htmlFor="category-is-public" className="flex items-start gap-2 text-sm text-slate-900">
            <input
              id="category-is-public"
              type="checkbox"
              className="mt-0.5 h-4 w-4 rounded-sm border-slate-300 accent-slate-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
              {...register('is_public')}
            />
            Public holiday (counts as a day off, excluded from leave)
          </label>
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
            <SubmitButton loading={pending}>{editing ? 'Save changes' : 'Add category'}</SubmitButton>
          </div>
        </form>
      </div>
    </div>
  )
}
