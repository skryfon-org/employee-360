import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link } from 'react-router-dom'
import { forgotPasswordSchema, type ForgotPasswordFormValues, AuthLayout, FormField, SubmitButton, InlineAlert } from '@employee360/ui'
import { useForgotPasswordMutation } from '../queries/authMutations'

export const FORGOT_SUCCESS_MESSAGE =
  'If an account exists for that email, we have sent password reset instructions.'

export default function ForgotPasswordPage() {
  const [done, setDone] = useState(false)
  const mutation = useForgotPasswordMutation()
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ForgotPasswordFormValues>({ resolver: zodResolver(forgotPasswordSchema) })

  // Enumeration-safe: the outcome is never reflected in the UI. Every API
  // response (success or error, including 4xx/5xx) yields the same message.
  const onSubmit = (values: ForgotPasswordFormValues) =>
    mutation.mutate({ email: values.email }, { onSettled: () => setDone(true) })

  return (
    <AuthLayout title="Forgot password">
      {done ? (
        <div className="flex flex-col gap-4">
          <InlineAlert tone="neutral">{FORGOT_SUCCESS_MESSAGE}</InlineAlert>
          <Link to="/login" className="inline-flex min-h-11 w-fit items-center text-sm text-slate-900 hover:text-slate-700">
            Back to sign in
          </Link>
        </div>
      ) : (
        <form onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-4">
          <p className="text-sm text-slate-600">
            Enter your email and we will send you a link to reset your password.
          </p>
          <FormField
            label="Email"
            type="email"
            autoComplete="username"
            error={errors.email?.message}
            {...register('email')}
          />
          <SubmitButton loading={mutation.isPending}>Send reset link</SubmitButton>
          <Link to="/login" className="inline-flex min-h-11 w-fit items-center text-sm text-slate-900 hover:text-slate-700">
            Back to sign in
          </Link>
        </form>
      )}
    </AuthLayout>
  )
}
