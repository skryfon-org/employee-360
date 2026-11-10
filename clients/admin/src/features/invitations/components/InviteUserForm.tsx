import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage, getErrorCode, EMAIL_DOMAIN_NOT_ALLOWED } from '@employee360/api-client'
import { inviteUserSchema, type InviteUserFormValues } from '../schemas/invitationSchemas'
import { useInviteUserMutation, useRolesQuery } from '../queries/invitationQueries'
import { Card, FormField, SubmitButton } from '@employee360/ui'
import { Link } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { ACTIVE_DEPARTMENTS_KEY, useActiveDepartmentsQuery } from '../../departments/queries/departmentQueries'
import { ACTIVE_POSITIONS_KEY, useActivePositionsQuery } from '../../positions/queries/positionQueries'
import { useToast } from '../../../hooks/useToast'

/** Backend error codes that refer to the selected department. */
const DEPARTMENT_ERROR_CODES = ['DEPARTMENT_INACTIVE', 'DEPARTMENT_NOT_FOUND']

/** Backend error codes that refer to the selected position. */
const POSITION_ERROR_CODES = ['POSITION_INACTIVE', 'POSITION_NOT_FOUND']

const EMPTY: InviteUserFormValues = {
  email: '',
  roleId: '',
  firstName: '',
  lastName: '',
  departmentId: '',
  positionId: '',
}

export function InviteUserForm({ onSuccess, onCancel }: { onSuccess?: () => void; onCancel?: () => void }) {
  const toast = useToast()
  const mutation = useInviteUserMutation()
  const roles = useRolesQuery()
  const roleOptions = roles.data ?? []
  const departments = useActiveDepartmentsQuery()
  const departmentOptions = departments.data ?? []
  const positions = useActivePositionsQuery()
  const positionOptions = positions.data ?? []
  const qc = useQueryClient()
  const {
    register,
    handleSubmit,
    reset,
    setError,
    setValue,
    formState: { errors },
  } = useForm<InviteUserFormValues>({ resolver: zodResolver(inviteUserSchema), defaultValues: EMPTY })

  const onSubmit = (v: InviteUserFormValues) =>
    mutation.mutate(
      {
        email: v.email,
        role_id: v.roleId,
        first_name: v.firstName || undefined,
        last_name: v.lastName || undefined,
        department_id: v.departmentId || undefined,
        position_id: v.positionId || undefined,
      },
      {
        onSuccess: () => {
          toast.success(`Invitation sent to ${v.email}.`)
          reset(EMPTY)
          onSuccess?.()
        },
        onError: (err) => {
          const message = getErrorMessage(err, 'Could not send invitation.')
          if (getErrorCode(err) === EMAIL_DOMAIN_NOT_ALLOWED) {
            setError('email', { type: 'server', message }, { shouldFocus: true })
            return
          }
          const code = getErrorCode(err)
          if (code && DEPARTMENT_ERROR_CODES.includes(code)) {
            setError('departmentId', { type: 'server', message }, { shouldFocus: true })
            // The selected department is stale: clear it and reload the options.
            setValue('departmentId', '')
            void qc.invalidateQueries({ queryKey: ACTIVE_DEPARTMENTS_KEY })
            return
          }
          if (code && POSITION_ERROR_CODES.includes(code)) {
            setError('positionId', { type: 'server', message }, { shouldFocus: true })
            setValue('positionId', '')
            void qc.invalidateQueries({ queryKey: ACTIVE_POSITIONS_KEY })
            return
          }
          toast.error(message)
        },
      },
    )

  return (
    <Card padded={false}>
      <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label="Invite user" className="flex flex-col">
        <div className="flex flex-col gap-6 p-4 sm:p-6">
          <div>
            <h2 className="text-base font-semibold text-ink">Invite a user</h2>
            <p className="mt-1 text-sm text-ink-muted">
              They will receive an email with a link to set up their account.
            </p>
          </div>
          <fieldset className="flex flex-col gap-4">
            <legend className="mb-3 text-xs font-medium text-ink-muted">Account</legend>
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField label="Email" type="email" error={errors.email?.message} {...register('email')} />
              <div className="flex flex-col gap-1">
                <label htmlFor="roleId" className="text-xs font-medium text-slate-900">
                  Role
                </label>
                <select
                  id="roleId"
                  aria-invalid={errors.roleId ? true : undefined}
                  aria-describedby={errors.roleId ? 'roleId-error' : undefined}
                  className="h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base sm:text-sm md:h-9 text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 aria-[invalid=true]:focus:ring-red-600"
                  {...register('roleId')}
                >
                  <option value="">{roles.isPending ? 'Loading roles...' : 'Select a role'}</option>
                  {roleOptions.map((r) => (
                    <option key={r.id} value={r.id}>
                      {r.name}
                    </option>
                  ))}
                </select>
                {roles.isError && (
                  <p role="alert" className="text-xs text-red-700">
                    Could not load roles.
                  </p>
                )}
                {errors.roleId && (
                  <p id="roleId-error" role="alert" className="text-xs text-red-700">
                    {errors.roleId.message}
                  </p>
                )}
              </div>
            </div>
          </fieldset>
          <div className="border-t border-line pt-6">
          <fieldset className="flex flex-col gap-4">
            <legend className="mb-3 text-xs font-medium text-ink-muted">Profile details</legend>
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField label="First name (optional)" error={errors.firstName?.message} {...register('firstName')} />
              <FormField label="Last name (optional)" error={errors.lastName?.message} {...register('lastName')} />
              <div className="flex flex-col gap-1">
                <label htmlFor="departmentId" className="text-xs font-medium text-slate-900">
                  Department (optional)
                </label>
                <select
                  id="departmentId"
                  aria-invalid={errors.departmentId ? true : undefined}
                  aria-describedby={errors.departmentId ? 'departmentId-error' : undefined}
                  disabled={departments.isPending}
                  className="h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base sm:text-sm md:h-9 text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 aria-[invalid=true]:focus:ring-red-600 disabled:bg-slate-100 disabled:text-slate-500"
                  {...register('departmentId')}
                >
                  <option value="">{departments.isPending ? 'Loading departments...' : 'No department selected'}</option>
                  {departmentOptions.map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.name}
                    </option>
                  ))}
                </select>
                {departments.isError && (
                  <p role="alert" className="text-xs text-red-700">
                    Could not load departments.{' '}
                    <button type="button" onClick={() => void departments.refetch()} className="underline">
                      Retry
                    </button>
                  </p>
                )}
                {departments.isSuccess && departmentOptions.length === 0 && (
                  <p className="text-xs text-slate-600">
                    No active departments yet.{' '}
                    <Link to="/departments" className="font-medium text-slate-900 underline">
                      Create one in Departments
                    </Link>
                    .
                  </p>
                )}
                {errors.departmentId && (
                  <p id="departmentId-error" role="alert" className="text-xs text-red-700">
                    {errors.departmentId.message}
                  </p>
                )}
              </div>
              <div className="flex flex-col gap-1">
                <label htmlFor="positionId" className="text-xs font-medium text-slate-900">
                  Position (optional)
                </label>
                <select
                  id="positionId"
                  aria-invalid={errors.positionId ? true : undefined}
                  aria-describedby={errors.positionId ? 'positionId-error' : undefined}
                  disabled={positions.isPending}
                  className="h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base sm:text-sm md:h-9 text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 aria-[invalid=true]:focus:ring-red-600 disabled:bg-slate-100 disabled:text-slate-500"
                  {...register('positionId')}
                >
                  <option value="">{positions.isPending ? 'Loading positions...' : 'No position selected'}</option>
                  {positionOptions.map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.name}
                    </option>
                  ))}
                </select>
                {positions.isError && (
                  <p role="alert" className="text-xs text-red-700">
                    Could not load positions.{' '}
                    <button type="button" onClick={() => void positions.refetch()} className="underline">
                      Retry
                    </button>
                  </p>
                )}
                {positions.isSuccess && positionOptions.length === 0 && (
                  <p className="text-xs text-slate-600">
                    No active positions yet.{' '}
                    <Link to="/positions" className="font-medium text-slate-900 underline">
                      Create one in Positions
                    </Link>
                    .
                  </p>
                )}
                {errors.positionId && (
                  <p id="positionId-error" role="alert" className="text-xs text-red-700">
                    {errors.positionId.message}
                  </p>
                )}
              </div>
            </div>
          </fieldset>
          </div>
        </div>
        <div className="flex flex-col-reverse gap-2 border-t border-line bg-surface-subtle p-4 sm:flex-row sm:justify-end sm:px-6">
          {onCancel && (
            <button
              type="button"
              onClick={onCancel}
              className="h-11 md:h-9 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 active:bg-slate-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
            >
              Cancel
            </button>
          )}
          <SubmitButton loading={mutation.isPending}>Send invitation</SubmitButton>
        </div>
      </form>
    </Card>
  )
}
