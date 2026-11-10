import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { resetPasswordSchema, type ResetPasswordFormValues, AuthLayout, FormField, SubmitButton, InlineAlert } from '@employee360/ui'
import { useResetPasswordMutation } from '../queries/authMutations'
import { clearSession } from '@employee360/api-client'
import { useAuthStore } from '../../../stores/authStore'

export default function ResetPasswordPage() {
  const [params] = useSearchParams()
  const token = params.get('token')
  const navigate = useNavigate()
  const mutation = useResetPasswordMutation()
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ResetPasswordFormValues>({ resolver: zodResolver(resetPasswordSchema) })

  if (!token) {
    return (
      <AuthLayout title="Reset password">
        <div className="flex flex-col gap-4">
          <InlineAlert tone="error">This reset link is invalid. Request a new one.</InlineAlert>
          <Link to="/forgot-password" className="inline-flex min-h-11 w-fit items-center text-sm text-slate-900 hover:text-slate-700">
            Request a new link
          </Link>
        </div>
      </AuthLayout>
    )
  }

  const onSubmit = (v: ResetPasswordFormValues) =>
    mutation.mutate(
      { token, new_password: v.newPassword },
      {
        onSuccess: () => {
          // Drop any active session and strip the token from URL/history.
          clearSession()
          useAuthStore.getState().clear()
          navigate('/login', { replace: true, state: { passwordReset: true } })
        },
      },
    )

  return (
    <AuthLayout title="Reset password">
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-4">
        {mutation.isError && (
          <InlineAlert tone="error">
            This reset link is invalid or has expired. Request a new one.
          </InlineAlert>
        )}
        <FormField
          label="New password"
          type="password"
          autoComplete="new-password"
          error={errors.newPassword?.message}
          {...register('newPassword')}
        />
        <FormField
          label="Confirm new password"
          type="password"
          autoComplete="new-password"
          error={errors.confirmPassword?.message}
          {...register('confirmPassword')}
        />
        <SubmitButton loading={mutation.isPending}>Reset password</SubmitButton>
        <Link to="/forgot-password" className="inline-flex min-h-11 w-fit items-center text-sm text-slate-900 hover:text-slate-700">
          Request a new link
        </Link>
      </form>
    </AuthLayout>
  )
}
