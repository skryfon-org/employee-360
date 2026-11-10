import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorCode, getErrorMessage, TENANT_ERROR_CODES, type TenantDomain } from '@employee360/api-client'
import { Card, FormField, InlineAlert, SubmitButton, RowActionsMenu } from '@employee360/ui'
import { addDomainSchema, type AddDomainFormValues } from '../schemas/tenantSchemas'
import { useAddDomainMutation, useRemoveDomainMutation, useUpdateDomainMutation } from '../queries/tenantQueries'
import { ConfirmDialog } from './ConfirmDialog'
import { useToast } from '../../../hooks/useToast'

const rowBtn =
  'h-11 md:h-8 rounded-sm border bg-white px-3 text-xs font-medium focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:border-slate-200 disabled:bg-slate-100 disabled:text-slate-500'
const neutralBtn = `${rowBtn} border-slate-300 text-slate-800 hover:bg-slate-100 focus-visible:ring-slate-900`

function isDomainFieldError(code: string | undefined) {
  return code === TENANT_ERROR_CODES.DOMAIN_ALREADY_EXISTS || code === TENANT_ERROR_CODES.INVALID_DOMAIN
}

const DOMAIN_HINT = 'Enter a full domain such as example.com'
const IN_USE_GUIDANCE = 'Move or deactivate those users from the Users page, and check pending Invitations, then try again.'
const inUseText = (message: string) => `${message} ${IN_USE_GUIDANCE}`

function DomainRow({
  domain,
  canRemove,
  onRemove,
  onInUse,
  onSaved,
}: {
  domain: TenantDomain
  canRemove: boolean
  onRemove: () => void
  onInUse: (message: string) => void
  onSaved: () => void
}) {
  const toast = useToast()
  const update = useUpdateDomainMutation()
  const [editing, setEditing] = useState(false)
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<AddDomainFormValues>({ resolver: zodResolver(addDomainSchema), defaultValues: { domain: domain.domain ?? '' } })

  const onSave = (v: AddDomainFormValues) =>
    update.mutate(
      { id: domain.id as string, domain: v.domain },
      {
        onSuccess: () => {
          toast.success(`Domain updated to ${v.domain}.`)
          onSaved()
          setEditing(false)
        },
        onError: (err) => {
          const message = getErrorMessage(err, 'Could not update domain.')
          if (getErrorCode(err) === TENANT_ERROR_CODES.DOMAIN_IN_USE) {
            onInUse(message)
            return
          }
          if (isDomainFieldError(getErrorCode(err))) {
            setError('domain', { type: 'server', message }, { shouldFocus: true })
            return
          }
          toast.error(message)
        },
      },
    )

  if (editing) {
    return (
      <li className="p-3 sm:px-4">
        <form onSubmit={handleSubmit(onSave)} noValidate aria-label={`Edit domain ${domain.domain}`} className="flex flex-col gap-3 sm:flex-row sm:items-start">
          <div className="min-w-0 flex-1">
            <FormField label="Domain" autoComplete="off" hint={DOMAIN_HINT} error={errors.domain?.message} {...register('domain')} />
          </div>
          <div className="grid grid-cols-2 gap-2 sm:mt-5 sm:flex sm:shrink-0 [&>button]:w-full sm:[&>button]:w-auto">
            <SubmitButton loading={update.isPending}>Save</SubmitButton>
            <button
              type="button"
              className={neutralBtn.replace('md:h-8', 'md:h-9')}
              onClick={() => {
                reset({ domain: domain.domain ?? '' })
                setEditing(false)
              }}
            >
              Cancel
            </button>
          </div>
        </form>
      </li>
    )
  }

  return (
    <li className="flex flex-col gap-3 p-3 text-sm text-slate-900 sm:flex-row sm:items-center sm:justify-between sm:gap-4 sm:px-4">
      <span className="min-w-0 break-all font-medium">{domain.domain}</span>
      <span className="grid grid-cols-2 gap-2 sm:flex sm:shrink-0">
        <RowActionsMenu
          label={`domain ${domain.domain}`}
          actions={[
            { label: 'Edit', onSelect: () => setEditing(true) },
            { label: 'Remove', danger: true, disabled: !canRemove, onSelect: onRemove },
          ]}
        />
      </span>
    </li>
  )
}

export function DomainsManager({ domains }: { domains: TenantDomain[] }) {
  const toast = useToast()
  const add = useAddDomainMutation()
  const remove = useRemoveDomainMutation()
  const [removing, setRemoving] = useState<TenantDomain | null>(null)
  const [cardAlert, setCardAlert] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<AddDomainFormValues>({ resolver: zodResolver(addDomainSchema), defaultValues: { domain: '' } })

  const onAdd = (v: AddDomainFormValues) =>
    add.mutate(v.domain, {
      onSuccess: () => {
        toast.success(`Domain ${v.domain} added.`)
        reset({ domain: '' })
        setCardAlert(null)
      },
      onError: (err) => {
        const message = getErrorMessage(err, 'Could not add domain.')
        if (isDomainFieldError(getErrorCode(err))) {
          setError('domain', { type: 'server', message }, { shouldFocus: true })
          return
        }
        toast.error(message)
      },
    })

  const onRemove = () => {
    if (!removing?.id) return
    const target = removing
    remove.mutate(target.id as string, {
      onSuccess: () => {
        toast.success(`Domain ${target.domain} removed.`)
        setCardAlert(null)
        setRemoving(null)
      },
      onError: (err) => {
        const message = getErrorMessage(err, 'Could not remove domain.')
        const code = getErrorCode(err)
        if (code === TENANT_ERROR_CODES.LAST_DOMAIN) {
          setRemoving(null)
          setCardAlert(message)
          return
        }
        if (code === TENANT_ERROR_CODES.DOMAIN_IN_USE) {
          setRemoving(null)
          setCardAlert(inUseText(message))
          return
        }
        toast.error(message)
      },
    })
  }

  return (
    <Card padded={false} aria-label="Domains" role="region">
      <div className="flex flex-col gap-4 p-4 sm:p-6">
        <div className="min-w-0">
          <h2 className="text-base font-semibold text-ink">Email domains</h2>
          <p className="mt-1 text-sm text-ink-muted">
            Users can only be invited with an email address on one of these domains. An organization must keep at least one domain.
          </p>
        </div>
        {cardAlert && (
          <div className="flex items-start gap-3">
            <div className="min-w-0 flex-1">
              <InlineAlert tone="error">{cardAlert}</InlineAlert>
            </div>
            <button type="button" className={neutralBtn} onClick={() => setCardAlert(null)}>
              Dismiss
            </button>
          </div>
        )}
        {domains.length === 0 ? (
          <p className="rounded-sm border border-dashed border-slate-300 bg-surface-subtle p-4 text-center text-sm text-ink-muted">No domains registered.</p>
        ) : (
          <ul className="divide-y divide-slate-200 rounded-sm border border-slate-200">
            {domains.map((d) => (
              <DomainRow
                key={d.id}
                domain={d}
                canRemove={domains.length > 1}
                onInUse={(m) => setCardAlert(inUseText(m))}
                onSaved={() => setCardAlert(null)}
                onRemove={() => {
                  remove.reset()
                  setRemoving(d)
                }}
              />
            ))}
          </ul>
        )}
        <form onSubmit={handleSubmit(onAdd)} noValidate aria-label="Add domain" className="flex flex-col gap-3 border-t border-line pt-4 sm:flex-row sm:items-start">
          <div className="min-w-0 flex-1">
            <FormField label="New domain" placeholder="acme.com" hint={DOMAIN_HINT} autoComplete="off" error={errors.domain?.message} {...register('domain')} />
          </div>
          <div className="sm:mt-5 sm:shrink-0 [&>button]:w-full sm:[&>button]:w-auto">
            <SubmitButton loading={add.isPending}>Add domain</SubmitButton>
          </div>
        </form>
      </div>
      {removing && (
        <ConfirmDialog
          title="Remove domain"
          confirmLabel="Remove"
          loading={remove.isPending}
          onCancel={() => setRemoving(null)}
          onConfirm={onRemove}
        >
          Remove <span className="break-all font-medium text-slate-900">{removing.domain}</span>? New invitations to addresses on
          this domain will be rejected.
        </ConfirmDialog>
      )}
    </Card>
  )
}
