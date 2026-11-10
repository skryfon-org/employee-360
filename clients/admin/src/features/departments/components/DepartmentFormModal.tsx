import { useRef } from 'react'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorCode, getErrorMessage } from '@employee360/api-client'
import { FormField, SubmitButton, Switch } from '@employee360/ui'
import {
  departmentFormSchema,
  DESCRIPTION_MAX,
  NAME_MAX,
  type DepartmentFormValues,
} from '../schemas/departmentSchemas'
import { useCreateDepartmentMutation, useUpdateDepartmentMutation, type Department } from '../queries/departmentQueries'
import { useDialogFocus } from '../../../hooks/useDialogFocus'
import { useToast } from '../../../hooks/useToast'

/** The backend uses CONFLICT for both duplicate name and in-use; on create/update it can only mean a duplicate name. */
const CONFLICT = 'CONFLICT'

export function DepartmentFormModal({
  department,
  onClose,
}: {
  /** Present when editing; absent when creating. */
  department?: Department
  onClose: () => void
}) {
  const toast = useToast()
  const create = useCreateDepartmentMutation()
  const update = useUpdateDepartmentMutation()
  const editing = Boolean(department)
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
  } = useForm<DepartmentFormValues>({
    resolver: zodResolver(departmentFormSchema),
    defaultValues: { name: department?.name ?? '', description: department?.description ?? '',
      is_active: department?.is_active ?? true,
    },
  })
  const descLength = [...(useWatch({ control, name: 'description' }) ?? '')].length

  const onSubmit = (v: DepartmentFormValues) => {
    const handlers = {
      onSuccess: () => {
        toast.success(editing ? `Department "${v.name}" updated.` : `Department "${v.name}" created.`)
        onClose()
      },
      onError: (err: unknown) => {
        if (getErrorCode(err) === CONFLICT) {
          setError('name', { type: 'server', message: 'A department with this name already exists.' })
          setFocus('name')
          return
        }
        toast.error(getErrorMessage(err, editing ? 'Could not update department.' : 'Could not create department.'))
      },
    }
    if (department?.id) update.mutate({ id: department.id, body: v }, handlers)
    else create.mutate(v, handlers)
  }

  const title = editing ? 'Edit department' : 'Create department'
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="department-form-title"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-md overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <h2 id="department-form-title" className="text-base font-semibold text-slate-900">
          {title}
        </h2>
        <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label={title} className="mt-4 flex flex-col gap-4">
          <FormField label="Name" autoComplete="off" error={errors.name?.message} {...register('name')} />
          <div className="flex flex-col gap-1">
            <label htmlFor="description" className="text-xs font-medium text-slate-900">
              Description
            </label>
            <textarea
              id="description"
              rows={4}
              aria-invalid={errors.description ? true : undefined}
              aria-describedby={errors.description ? 'description-error' : 'description-hint'}
              className="w-full rounded-sm border border-slate-300 bg-white px-3 py-2 text-base text-slate-900 placeholder:text-slate-500 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 aria-[invalid=true]:focus:ring-red-600 sm:text-sm"
              {...register('description')}
            />
            {errors.description ? (
              <p id="description-error" role="alert" className="text-xs text-red-700">
                {errors.description.message}
              </p>
            ) : (
              <p id="description-hint" className="text-xs text-slate-600">
                {descLength}/{DESCRIPTION_MAX} characters. Name up to {NAME_MAX}.
              </p>
            )}
          </div>
          <Controller
            control={control}
            name="is_active"
            render={({ field }) => (
              <Switch
                id="is_active"
                ref={field.ref}
                name={field.name}
                checked={field.value}
                onChange={field.onChange}
                onBlur={field.onBlur}
                label="Active"
                description="Inactive departments cannot be assigned to new invitations."
              />
            )}
          />
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
            <SubmitButton loading={pending}>{editing ? 'Save changes' : 'Create'}</SubmitButton>
          </div>
        </form>
      </div>
    </div>
  )
}
