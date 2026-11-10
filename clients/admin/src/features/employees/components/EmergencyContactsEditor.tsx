import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorMessage } from '@employee360/api-client'
import { FormField, SubmitButton, RowActionsMenu } from '@employee360/ui'
import { ConfirmDialog } from '../../tenants/components/ConfirmDialog'
import { useToast } from '../../../hooks/useToast'
import {
  useCreateContactMutation,
  useDeleteContactMutation,
  useUpdateContactMutation,
  type ContactInput,
  type EmergencyContact,
} from '../queries/employeeQueries'
import { MAX_CONTACTS, contactSchema, type ContactValues } from '../schemas/employeeSchemas'
import { Section, secondaryButtonCls } from './fields'

function toInput(v: ContactValues): ContactInput {
  return {
    name: v.name.trim(),
    relationship: v.relationship.trim(),
    phone: v.phone.trim(),
    alternate_phone: v.alternate_phone.trim(),
    email: v.email.trim(),
    is_primary: v.is_primary,
  }
}

function ContactForm({
  employeeId,
  contact,
  onDone,
}: {
  employeeId: string
  contact?: EmergencyContact
  onDone: () => void
}) {
  const toast = useToast()
  const create = useCreateContactMutation()
  const update = useUpdateContactMutation()
  const pending = create.isPending || update.isPending
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
      is_primary: contact?.is_primary ?? false,
    },
  })

  const onSubmit = (v: ContactValues) => {
    const handlers = {
      onSuccess: () => {
        toast.success(contact ? 'Emergency contact updated.' : 'Emergency contact added.')
        onDone()
      },
      onError: (err: unknown) => toast.error(getErrorMessage(err, 'Could not save emergency contact.')),
    }
    if (contact?.id) update.mutate({ id: employeeId, contactId: contact.id, body: toInput(v) }, handlers)
    else create.mutate({ id: employeeId, body: toInput(v) }, handlers)
  }

  const label = contact ? 'Edit emergency contact' : 'Add emergency contact'
  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      noValidate
      aria-label={label}
      className="flex flex-col gap-4 rounded-sm border border-slate-300 bg-slate-100 p-4"
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <FormField label="Name" required id={`${pfx}-name`} error={errors.name?.message} {...register('name')} />
        <FormField label="Relationship" required id={`${pfx}-relationship`} error={errors.relationship?.message} {...register('relationship')} />
        <FormField label="Phone" required type="tel" id={`${pfx}-phone`} error={errors.phone?.message} {...register('phone')} />
        <FormField label="Alternate phone" type="tel" id={`${pfx}-alternate_phone`} error={errors.alternate_phone?.message} {...register('alternate_phone')} />
        <FormField label="Email" type="email" id={`${pfx}-email`} error={errors.email?.message} {...register('email')} />
      </div>
      <label className="flex items-center gap-2 text-sm text-slate-900">
        <input type="checkbox" className="h-4 w-4 accent-slate-900" {...register('is_primary')} />
        Primary contact
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

export function EmergencyContactsEditor({
  employeeId,
  contacts,
  editing = false,
}: {
  employeeId: string
  contacts: EmergencyContact[]
  /** View mode (default) lists contacts only; edit mode enables add / edit / remove. */
  editing?: boolean
}) {
  const toast = useToast()
  const del = useDeleteContactMutation()
  const [adding, setAdding] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<EmergencyContact | null>(null)
  const atLimit = contacts.length >= MAX_CONTACTS

  return (
    <Section title="Emergency contacts" description={`Up to ${MAX_CONTACTS} contacts, one of them primary.`}>
      {contacts.length === 0 && !adding && <p className="text-sm text-slate-600">No emergency contacts yet.</p>}
      <ul className="flex flex-col gap-3">
        {contacts.map((c) =>
          editing && editingId === c.id ? (
            <li key={c.id}>
              <ContactForm employeeId={employeeId} contact={c} onDone={() => setEditingId(null)} />
            </li>
          ) : (
            <li
              key={c.id}
              className="flex flex-col gap-3 rounded-sm border border-slate-200 bg-white p-4 sm:flex-row sm:items-start sm:justify-between"
            >
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
              </div>
              {editing && (
              <RowActionsMenu
                label={`contact ${c.name}`}
                actions={[
                  { label: 'Edit', onSelect: () => setEditingId(c.id ?? null) },
                  {
                    label: 'Remove',
                    danger: true,
                    onSelect: () => {
                      del.reset()
                      setDeleting(c)
                    },
                  },
                ]}
              />
              )}
            </li>
          ),
        )}
      </ul>
      {editing && (adding ? (
        <ContactForm employeeId={employeeId} onDone={() => setAdding(false)} />
      ) : (
        <div>
          <button type="button" className={secondaryButtonCls} disabled={atLimit} onClick={() => setAdding(true)}>
            Add emergency contact
          </button>
          {atLimit && <p className="mt-1 text-xs text-slate-600">The maximum of {MAX_CONTACTS} contacts has been reached.</p>}
        </div>
      ))}
      {deleting && (
        <ConfirmDialog
          title="Remove emergency contact?"
          confirmLabel="Remove"
          loading={del.isPending}
          error={del.isError ? getErrorMessage(del.error, 'Could not remove emergency contact.') : undefined}
          onCancel={() => setDeleting(null)}
          onConfirm={() =>
            del.mutate(
              { id: employeeId, contactId: deleting.id as string },
              {
                onSuccess: () => {
                  toast.success('Emergency contact removed.')
                  setDeleting(null)
                },
              },
            )
          }
        >
          {deleting.name} will be removed from this employee&apos;s emergency contacts.
        </ConfirmDialog>
      )}
    </Section>
  )
}
