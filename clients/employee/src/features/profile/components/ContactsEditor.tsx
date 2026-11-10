import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage } from '@employee360/api-client'
import { FormField, InlineAlert, SubmitButton, RowActionsMenu } from '@employee360/ui'
import {
  useCreateContactMutation,
  useDeleteContactMutation,
  useUpdateContactMutation,
  type ContactInput,
  type EmergencyContact,
} from '../queries/profileQueries'
import { MAX_CONTACTS, contactSchema, type ContactValues } from '../schemas/profileSchemas'
import { Section, dangerButtonCls, secondaryButtonCls } from './fields'

function ContactForm({
  contact,
  isFirst,
  refreshMe,
  onDone,
}: {
  contact?: EmergencyContact
  /** No contacts exist yet: the first contact becomes primary automatically. */
  isFirst: boolean
  refreshMe: boolean
  onDone: () => void
}) {
  const create = useCreateContactMutation(refreshMe)
  const update = useUpdateContactMutation(refreshMe)
  const pending = create.isPending || update.isPending
  const error = create.error ?? update.error
  const pfx = `contact-${contact?.id ?? 'new'}`
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ContactValues>({
    resolver: zodResolver(contactSchema),
    defaultValues: {
      name: contact?.name ?? '',
      relationship: contact?.relationship ?? '',
      phone: contact?.phone ?? '',
      alternate_phone: contact?.alternate_phone ?? '',
      email: contact?.email ?? '',
      is_primary: isFirst || (contact?.is_primary ?? false),
    },
  })

  const onSubmit = (v: ContactValues) => {
    const body: ContactInput = {
      name: v.name.trim(),
      relationship: v.relationship.trim(),
      phone: v.phone.trim(),
      alternate_phone: v.alternate_phone.trim(),
      email: v.email.trim(),
      is_primary: v.is_primary,
    }
    if (contact?.id) update.mutate({ contactId: contact.id, body }, { onSuccess: onDone })
    else create.mutate(body, { onSuccess: onDone })
  }

  const label = contact ? 'Edit emergency contact' : 'Add emergency contact'
  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate aria-label={label} className="flex flex-col gap-4 rounded-sm border border-slate-300 bg-slate-100 p-4">
      {error && <InlineAlert tone="error">{getErrorMessage(error, 'Could not save emergency contact.')}</InlineAlert>}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <FormField id={`${pfx}-name`} label="Name" required error={errors.name?.message} {...register('name')} />
        <FormField id={`${pfx}-relationship`} label="Relationship" required error={errors.relationship?.message} {...register('relationship')} />
        <FormField id={`${pfx}-phone`} label="Phone" required type="tel" error={errors.phone?.message} {...register('phone')} />
        <FormField id={`${pfx}-alternate_phone`} label="Alternate phone" type="tel" error={errors.alternate_phone?.message} {...register('alternate_phone')} />
        <FormField id={`${pfx}-email`} label="Email" type="email" error={errors.email?.message} {...register('email')} />
      </div>
      <label className="flex items-center gap-2 text-sm text-slate-900">
        <input type="checkbox" disabled={isFirst} className="h-4 w-4 accent-slate-900" {...register('is_primary')} />
        Primary contact
        {isFirst && <span className="text-xs text-slate-600">(your first contact is primary)</span>}
      </label>
      <div className="flex justify-end gap-2">
        <button type="button" className={secondaryButtonCls} onClick={onDone} disabled={pending}>
          Cancel
        </button>
        <SubmitButton loading={pending}>{contact ? 'Save contact' : 'Add contact'}</SubmitButton>
      </div>
    </form>
  )
}

/** Emergency contacts: up to 10, exactly one primary, at least one required (the last cannot be removed). */
export function ContactsEditor({
  contacts,
  refreshMe,
  editing = true,
}: {
  contacts: EmergencyContact[]
  refreshMe: boolean
  /** False renders a read-only list (profile page view mode); the wizard always edits. */
  editing?: boolean
}) {
  const del = useDeleteContactMutation(refreshMe)
  const [adding, setAdding] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [removingId, setRemovingId] = useState<string | null>(null)
  const atLimit = contacts.length >= MAX_CONTACTS
  const onlyOne = contacts.length <= 1

  return (
    <Section title="Emergency contacts" description={`At least one is required. Up to ${MAX_CONTACTS}, one of them primary.`}>
      {contacts.length === 0 && !adding && (
        <p className="text-sm text-slate-600">No emergency contacts yet. Add at least one to continue.</p>
      )}
      <ul className="flex flex-col gap-3">
        {contacts.map((c) =>
          editing && editingId === c.id ? (
            <li key={c.id}>
              <ContactForm contact={c} isFirst={false} refreshMe={refreshMe} onDone={() => setEditingId(null)} />
            </li>
          ) : (
            <li key={c.id} className="flex flex-col gap-3 rounded-sm border border-slate-200 bg-white p-4 sm:flex-row sm:items-start sm:justify-between">
              <div className="min-w-0 text-sm">
                <p className="font-medium text-slate-900">
                  {c.name}
                  {c.is_primary && (
                    <span className="ml-2 inline-flex h-6 items-center rounded-sm border border-green-300 bg-green-50 px-2 text-xs font-medium text-green-700">
                      Primary
                    </span>
                  )}
                </p>
                <p className="text-slate-600">
                  {c.relationship} · {c.phone}
                  {c.alternate_phone ? ` · ${c.alternate_phone}` : ''}
                </p>
                {c.email && <p className="break-all text-slate-600">{c.email}</p>}
                {removingId === c.id && del.isError && (
                  <p role="alert" className="mt-1 text-xs text-red-700">{getErrorMessage(del.error, 'Could not remove contact.')}</p>
                )}
              </div>
              {editing && (
              <div className="flex flex-wrap gap-2">
                {removingId === c.id ? (
                  <>
                    <button
                      type="button"
                      className={dangerButtonCls}
                      disabled={del.isPending}
                      aria-label={`Confirm remove ${c.name}`}
                      onClick={() => del.mutate(c.id as string, { onSuccess: () => setRemovingId(null) })}
                    >
                      Confirm remove
                    </button>
                    <button type="button" className={secondaryButtonCls} onClick={() => setRemovingId(null)} disabled={del.isPending}>
                      Cancel
                    </button>
                  </>
                ) : (
                  <>
                    <RowActionsMenu
                      label={`contact ${c.name}`}
                      actions={[
                        { label: 'Edit', onSelect: () => setEditingId(c.id ?? null) },
                        {
                          label: 'Remove',
                          danger: true,
                          disabled: onlyOne,
                          onSelect: () => {
                            del.reset()
                            setRemovingId(c.id ?? null)
                          },
                        },
                      ]}
                    />
                  </>
                )}
              </div>
              )}
            </li>
          ),
        )}
      </ul>
      {editing && (adding ? (
        <ContactForm isFirst={contacts.length === 0} refreshMe={refreshMe} onDone={() => setAdding(false)} />
      ) : (
        <div>
          <button type="button" className={secondaryButtonCls} disabled={atLimit} onClick={() => setAdding(true)}>
            Add emergency contact
          </button>
          {atLimit && <p className="mt-1 text-xs text-slate-600">The maximum of {MAX_CONTACTS} contacts has been reached.</p>}
          {onlyOne && contacts.length === 1 && (
            <p className="mt-1 text-xs text-slate-600">Add another contact before removing your only one.</p>
          )}
        </div>
      ))}
    </Section>
  )
}
