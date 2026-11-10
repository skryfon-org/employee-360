import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorCode, getErrorMessage, TENANT_ERROR_CODES } from '@employee360/api-client'
import { Card, FormField, SubmitButton } from '@employee360/ui'
import { renameTenantSchema, type RenameTenantFormValues } from '../schemas/tenantSchemas'
import { useRenameTenantMutation } from '../queries/tenantQueries'
import { useToast } from '../../../hooks/useToast'

export function RenameTenantForm({ name }: { name: string }) {
  const toast = useToast()
  const mutation = useRenameTenantMutation()
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<RenameTenantFormValues>({ resolver: zodResolver(renameTenantSchema), defaultValues: { name } })

  // Keep the field in sync after a successful save / refetch.
  useEffect(() => reset({ name }), [name, reset])

  const onSubmit = (v: RenameTenantFormValues) =>
    mutation.mutate(v.name, {
      onSuccess: () => toast.success('Organization renamed.'),
      onError: (err) => {
        const message = getErrorMessage(err, 'Could not rename organization.')
        if (getErrorCode(err) === TENANT_ERROR_CODES.INVALID_NAME) {
          setError('name', { type: 'server', message }, { shouldFocus: true })
          return
        }
        toast.error(message)
      },
    })

  return (
    <Card padded={false}>
      <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label="Rename organization" className="flex flex-col gap-4 p-4 sm:p-6">
        <div>
          <h2 className="text-base font-semibold text-ink">Organization name</h2>
          <p className="mt-1 text-sm text-ink-muted">This is how your organization appears to administrators and employees.</p>
        </div>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
          <div className="min-w-0 flex-1">
            <FormField label="Name" error={errors.name?.message} {...register('name')} />
          </div>
          <div className="sm:mt-5 sm:shrink-0 [&>button]:w-full sm:[&>button]:w-auto">
            <SubmitButton loading={mutation.isPending}>Save name</SubmitButton>
          </div>
        </div>
      </form>
    </Card>
  )
}
