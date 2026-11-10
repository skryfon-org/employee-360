import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { loginSchema, type LoginFormValues, AuthLayout, FormField, SubmitButton, InlineAlert } from '@employee360/ui'
import { NotEmployeeError, useLoginMutation } from '../queries/authMutations'
import { isAuthenticated, useAuthStore } from '../../../stores/authStore'

const GENERIC_ERROR = 'Invalid email or password.'
const UNAVAILABLE_ERROR = 'Unable to sign in. Please try again.'

function loginErrorMessage(error: unknown): string {
  if (error instanceof NotEmployeeError) return GENERIC_ERROR
  const status = (error as { response?: { status?: number } } | null)?.response?.status
  return status === 401 || status === 403 ? GENERIC_ERROR : UNAVAILABLE_ERROR
}

export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const authed = useAuthStore(isAuthenticated)
  const mutation = useLoginMutation()
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormValues>({ resolver: zodResolver(loginSchema) })

  const state = location.state as { from?: string; passwordReset?: boolean; invitationAccepted?: boolean } | null
  const from = state?.from ?? '/'
  const passwordReset = Boolean(state?.passwordReset)
  const invitationAccepted = Boolean(state?.invitationAccepted)
  if (authed && !mutation.isPending) return <Navigate to={from} replace />

  const onSubmit = (values: LoginFormValues) =>
    mutation.mutate(values, { onSuccess: () => navigate(from, { replace: true }) })

  return (
    <AuthLayout title="Sign in to Employee Portal">
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-4">
        {mutation.isError && <InlineAlert tone="error">{loginErrorMessage(mutation.error)}</InlineAlert>}
        {!mutation.isError && passwordReset && (
          <InlineAlert tone="neutral">Your password has been reset. You can now sign in.</InlineAlert>
        )}
        {!mutation.isError && invitationAccepted && (
          <InlineAlert tone="neutral">Your invitation was accepted. You can now sign in.</InlineAlert>
        )}
        <FormField
          label="Email"
          type="email"
          autoComplete="username"
          error={errors.email?.message}
          {...register('email')}
        />
        <FormField
          label="Password"
          type="password"
          autoComplete="current-password"
          error={errors.password?.message}
          {...register('password')}
        />
        <SubmitButton loading={mutation.isPending}>Sign in</SubmitButton>
        <Link to="/forgot-password" className="inline-flex min-h-11 w-fit items-center text-sm text-slate-900 hover:text-slate-700">
          Forgot password?
        </Link>
      </form>
    </AuthLayout>
  )
}
