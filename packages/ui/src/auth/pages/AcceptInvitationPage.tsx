import { useEffect, type ReactNode } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useSearchParams } from 'react-router-dom'
import { acceptInvitationSchema, type AcceptInvitationFormValues } from '../schemas/authSchemas'
import {
  acceptInvitation,
  validateInvitation,
  type AcceptInvitationRequest,
  type InvitationApp,
  INVITATION_ERROR_CODES,
  clearSession,
  getErrorCode,
  getErrorMessage,
  resolveInvitationRedirect,
} from '@employee360/api-client'
import { AuthLayout } from '../components/AuthLayout'
import { FormField } from '../components/FormField'
import { SubmitButton } from '../components/SubmitButton'
import { InlineAlert } from '../components/InlineAlert'
import { Skeleton, SkeletonRegion } from '../../layout/Skeleton'

export interface AcceptInvitationPageProps {
  /** Which portal is rendering this page. */
  currentApp: InvitationApp
  /** Label of the post-accept "sign in" button, e.g. "Sign in to Admin". */
  signInLabel: string
  /** Other portal's name as shown in the redirect/unconfigured notices, e.g. "Employee Portal". */
  otherPortalName: string
  /** Other portal's name as shown in the "already accepted" message (casing differs per app). */
  otherPortalNameInAcceptedHint: string
  /** Base URLs of the portals, supplied by the host app (this package never reads import.meta.env). */
  adminAppUrl?: string
  employeeAppUrl?: string
  /** Called after the invitation is accepted, so the host app can clear its own session state. */
  onAccepted?: () => void
}

type InvitationProblem = {
  title: string
  message: string
  /** Show a primary "Sign in" button (invitation already accepted). */
  signIn?: boolean
}

function acceptedMessage(otherPortalName: string) {
  return `This invitation has already been accepted. You can sign in with your password. If you were invited to the ${otherPortalName}, sign in there instead.`
}

const INVITATION_PROBLEMS = {
  INVALID_TOKEN: {
    title: 'Invalid invitation link',
    message: 'This invitation link is invalid. Please check the link in your invitation email and try again.',
  },
  EXPIRED: {
    title: 'Invitation expired',
    message: 'This invitation has expired. Ask your administrator to resend the invitation.',
  },
  REVOKED: {
    title: 'Invitation revoked',
    message: 'This invitation has been revoked. Please contact your administrator.',
  },
  ACCEPTED: {
    title: 'Invitation already accepted',
    message: '',
    signIn: true,
  },
} satisfies Record<string, InvitationProblem>

/** Map a backend error code to a distinct invitation problem state (never by message text). */
function problemForError(error: unknown, otherPortalName: string): InvitationProblem | null {
  switch (getErrorCode(error)) {
    case INVITATION_ERROR_CODES.INVALID_TOKEN:
      return INVITATION_PROBLEMS.INVALID_TOKEN
    case INVITATION_ERROR_CODES.EXPIRED:
      return INVITATION_PROBLEMS.EXPIRED
    case INVITATION_ERROR_CODES.REVOKED:
      return INVITATION_PROBLEMS.REVOKED
    case INVITATION_ERROR_CODES.ACCEPTED:
      return { ...INVITATION_PROBLEMS.ACCEPTED, message: acceptedMessage(otherPortalName) }
    default:
      return null
  }
}

function ProblemView({ problem }: { problem: InvitationProblem }) {
  return (
    <AuthLayout title={problem.title}>
      <div className="flex flex-col gap-4">
        <InlineAlert tone="error">{problem.message}</InlineAlert>
        {problem.signIn ? (
          <Link
            to="/login"
            className="inline-flex h-11 md:h-9 items-center justify-center gap-2 rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950"
          >
            Sign in
          </Link>
        ) : (
          <Link to="/login" className="inline-flex min-h-11 w-fit items-center text-sm text-slate-900 hover:text-slate-700">
            Return to sign in
          </Link>
        )}
      </div>
    </AuthLayout>
  )
}

function Notice({ title, children }: { title: string; children: ReactNode }) {
  return (
    <AuthLayout title={title}>
      <InlineAlert tone="neutral">{children}</InlineAlert>
    </AuthLayout>
  )
}

export function AcceptInvitationPage({
  currentApp,
  signInLabel,
  otherPortalName,
  otherPortalNameInAcceptedHint,
  adminAppUrl,
  employeeAppUrl,
  onAccepted,
}: AcceptInvitationPageProps) {
  const [params] = useSearchParams()
  const token = params.get('token')
  const validateQuery = useQuery({
    queryKey: ['invitations', 'validate', token],
    queryFn: ({ signal }) => validateInvitation(token!, signal),
    enabled: Boolean(token),
    retry: false,
  })
  const mutation = useMutation({ mutationFn: (body: AcceptInvitationRequest) => acceptInvitation(body) })
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<AcceptInvitationFormValues>({ resolver: zodResolver(acceptInvitationSchema) })

  const invitation = validateQuery.data
  const redirect =
    token && invitation
      ? resolveInvitationRedirect({
          role: invitation.role,
          currentApp,
          token,
          adminAppUrl,
          employeeAppUrl,
        })
      : null
  const redirectUrl = redirect?.kind === 'redirect' ? redirect.url : null

  useEffect(() => {
    if (redirectUrl) window.location.replace(redirectUrl)
  }, [redirectUrl])

  if (!token) return <ProblemView problem={INVITATION_PROBLEMS.INVALID_TOKEN} />

  if (validateQuery.isLoading) {
    return (
      <AuthLayout title="Verifying invitation">
        <SkeletonRegion label="Verifying your invitation link…" className="flex flex-col gap-4">
          <div className="flex flex-col gap-1"><Skeleton className="h-4 w-24" /><Skeleton className="h-9 w-full" /></div>
          <div className="flex flex-col gap-1"><Skeleton className="h-4 w-24" /><Skeleton className="h-9 w-full" /></div>
          <Skeleton className="h-9 w-full" />
        </SkeletonRegion>
      </AuthLayout>
    )
  }

  if (validateQuery.isError) {
    const problem = problemForError(validateQuery.error, otherPortalNameInAcceptedHint)
    if (problem) return <ProblemView problem={problem} />
    return (
      <AuthLayout title="Accept invitation">
        <div className="flex flex-col gap-4">
          <InlineAlert tone="error">
            {getErrorMessage(validateQuery.error, 'Unable to verify this invitation. Please try again.')}
          </InlineAlert>
          <Link to="/login" className="inline-flex min-h-11 w-fit items-center text-sm text-slate-900 hover:text-slate-700">
            Return to sign in
          </Link>
        </div>
      </AuthLayout>
    )
  }

  if (redirect?.kind === 'redirect') {
    return (
      <Notice title="Redirecting">
        This invitation is for the {otherPortalName}. Redirecting you now...{' '}
        <a href={redirect.url} className="font-medium">
          Continue
        </a>
      </Notice>
    )
  }

  if (redirect?.kind === 'unconfigured') {
    return (
      <Notice title={`Open the ${otherPortalName}`}>
        This invitation is for the {otherPortalName}; open the link from your invitation email in that app
        (the {otherPortalName} address is not configured here).
      </Notice>
    )
  }

  if (mutation.isSuccess) {
    return (
      <AuthLayout title="Invitation accepted">
        <div className="flex flex-col gap-4">
          <InlineAlert tone="neutral">
            Your invitation has been accepted and your password has been set. You can now sign in to your account.
          </InlineAlert>
          <div className="flex flex-col gap-2 pt-2">
            <Link
              to="/login"
              state={{ invitationAccepted: true }}
              className="inline-flex h-11 md:h-9 items-center justify-center gap-2 rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950"
            >
              {signInLabel}
            </Link>
          </div>
        </div>
      </AuthLayout>
    )
  }

  // A race (e.g. accepted/revoked/expired between validate and submit) gets the same distinct state.
  const acceptProblem = mutation.isError ? problemForError(mutation.error, otherPortalNameInAcceptedHint) : null
  if (acceptProblem) return <ProblemView problem={acceptProblem} />

  const onSubmit = (v: AcceptInvitationFormValues) => {
    mutation.mutate(
      { token, password: v.password },
      {
        onSuccess: () => {
          clearSession()
          onAccepted?.()
        },
      },
    )
  }

  return (
    <AuthLayout title="Set your password">
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-4">
        {mutation.isError && (
          <InlineAlert tone="error">
            {getErrorMessage(mutation.error, 'Unable to accept invitation. Please try again.')}
          </InlineAlert>
        )}
        <p className="text-xs text-slate-600">
          {invitation?.email ? (
            <>
              Setting a password for <span className="font-semibold text-slate-900">{invitation.email}</span>
            </>
          ) : (
            'Set a password to complete your account setup and accept the invitation.'
          )}
        </p>
        <FormField
          label="Password"
          type="password"
          autoComplete="new-password"
          error={errors.password?.message}
          {...register('password')}
        />
        <FormField
          label="Confirm password"
          type="password"
          autoComplete="new-password"
          error={errors.confirmPassword?.message}
          {...register('confirmPassword')}
        />
        <SubmitButton loading={mutation.isPending}>Set password & accept</SubmitButton>
        <Link to="/login" className="inline-flex min-h-11 w-fit items-center text-sm text-slate-900 hover:text-slate-700">
          Already accepted? Sign in
        </Link>
      </form>
    </AuthLayout>
  )
}
