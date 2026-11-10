import { zodResolver } from '@hookform/resolvers/zod'
import { FormField, InlineAlert, SubmitButton } from '@employee360/ui'
import { useEffect, useMemo, useRef, type KeyboardEvent } from 'react'
import { useForm } from 'react-hook-form'
import { ACTIVITY_CATEGORIES, buildEntrySchema, parseTarget, type EntryFormValues } from '../schemas/statusLogSchemas'
import type { CreateStatusLog, StatusLogEntry } from '../queries/statusLogQueries'
import { humanize, targetOf } from '../targets'
import { continueList, toggleList, type ListKind, type NotesState } from '../notesFormat'

export interface ProjectOption {
  id: string
  label: string
  /** Allocation window (ISO dates); end omitted = open-ended. */
  startDate?: string
  endDate?: string
}


interface Props {
  projects: ProjectOption[]
  /** Day total excluding the entry being edited. */
  otherHours: number
  entry?: StatusLogEntry
  saving: boolean
  /** Server/validation error shown above the buttons. */
  error?: string | null
  onSubmit: (body: Pick<CreateStatusLog, 'project_id' | 'activity_category' | 'hours' | 'notes'>) => void
  onCancel?: () => void
  /** Reports RHF's dirty state so a wrapper can guard closing. */
  onDirtyChange?: (dirty: boolean) => void
}

export function EntryForm({ projects, otherHours, entry, saving, error, onSubmit, onCancel, onDirtyChange }: Props) {
  const schema = useMemo(() => buildEntrySchema(otherHours), [otherHours])
  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors, isDirty },
  } = useForm<EntryFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { target: entry ? targetOf(entry) : '', hours: entry?.hours != null ? String(entry.hours) : '', notes: entry?.notes ?? '' },
  })

  useEffect(() => {
    onDirtyChange?.(isDirty)
  }, [isDirty, onDirtyChange])

  const notesField = register('notes')
  const notesRef = useRef<HTMLTextAreaElement | null>(null)

  const applyNotes = (next: NotesState) => {
    const el = notesRef.current
    if (!el) return
    el.value = next.text
    setValue('notes', next.text, { shouldDirty: true, shouldValidate: true })
    el.focus()
    el.setSelectionRange(next.selectionStart, next.selectionEnd)
  }

  const onListClick = (kind: ListKind) => {
    const el = notesRef.current
    if (!el) return
    applyNotes(toggleList(el.value, el.selectionStart, el.selectionEnd, kind))
  }

  const onNotesKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key !== 'Enter' || e.shiftKey || e.ctrlKey || e.metaKey || e.altKey || e.nativeEvent.isComposing) return
    const el = e.currentTarget
    const next = continueList(el.value, el.selectionStart, el.selectionEnd)
    if (!next) return
    e.preventDefault()
    applyNotes(next)
  }

  const options = [...projects]
  if (entry?.project_id && !options.some((p) => p.id === entry.project_id)) {
    options.push({ id: entry.project_id, label: entry.project_name ?? entry.project_id })
  }
  const idPrefix = entry ? `edit-${entry.id}` : 'new'

  const submit = handleSubmit((v) => {
    const t = parseTarget(v.target)
    if (!t) return
    onSubmit({ ...t, hours: Number(v.hours), notes: v.notes.trim() })
  })

  const onFormKeyDown = (e: KeyboardEvent<HTMLFormElement>) => {
    if (e.key !== 'Enter' || !(e.ctrlKey || e.metaKey) || e.nativeEvent.isComposing) return
    e.preventDefault()
    if (!saving) void submit()
  }

  return (
    <form onSubmit={submit} onKeyDown={onFormKeyDown} noValidate className="flex flex-col gap-4">
      <div className="grid gap-4 sm:grid-cols-[2fr_1fr]">
        <div className="flex flex-col gap-1">
          <label htmlFor={`${idPrefix}-target`} className="text-xs font-medium text-slate-900">
            Project or category
            <span className="text-red-600" aria-hidden="true">
              {' '}
              *
            </span>
          </label>
          <select
            id={`${idPrefix}-target`}
            aria-required="true"
            aria-invalid={errors.target ? true : undefined}
            className="h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 sm:text-sm md:h-9"
            {...register('target')}
          >
            <option value="">Select…</option>
            {options.length > 0 && (
              <optgroup label="Projects">
                {options.map((p) => (
                  <option key={p.id} value={`project:${p.id}`}>
                    {p.label}
                  </option>
                ))}
              </optgroup>
            )}
            <optgroup label="Other activity">
              {ACTIVITY_CATEGORIES.map((c) => (
                <option key={c} value={`category:${c}`}>
                  {humanize(c)}
                </option>
              ))}
            </optgroup>
          </select>
          {options.length === 0 && (
            <p className="text-xs text-slate-600">No projects are allocated to you on this date. Other activity categories are still available.</p>
          )}
          {errors.target && (
            <p role="alert" className="text-xs text-red-700">
              {errors.target.message}
            </p>
          )}
        </div>
        <FormField id={`${idPrefix}-hours`} label="Hours" required inputMode="decimal" error={errors.hours?.message} {...register('hours')} />
      </div>
      <div className="flex flex-col gap-1">
        <label htmlFor={`${idPrefix}-notes`} className="text-xs font-medium text-slate-900">
          Notes
        </label>
        <div className="flex gap-2">
          {(
            [
              ['bullet', 'Bullet list'],
              ['numbered', 'Numbered list'],
            ] as const
          ).map(([kind, label]) => (
            <button
              key={kind}
              type="button"
              aria-label={label}
              onClick={() => onListClick(kind)}
              className="h-8 rounded-sm border border-slate-300 bg-white px-2 text-xs font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
            >
              {label}
            </button>
          ))}
        </div>
        <textarea
          id={`${idPrefix}-notes`}
          rows={3}
          className="w-full rounded-sm border border-slate-300 bg-white px-3 py-2 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm"
          {...notesField}
          ref={(el) => {
            notesField.ref(el)
            notesRef.current = el
          }}
          onKeyDown={onNotesKeyDown}
        />
      </div>
      {error && <InlineAlert tone="error">{error}</InlineAlert>}
      <div className="flex items-center justify-end gap-3">
        <p className="mr-auto hidden text-xs text-slate-600 sm:block">Ctrl+Enter to save</p>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
          >
            Cancel
          </button>
        )}
        <SubmitButton loading={saving}>{entry ? 'Save changes' : 'Save entry'}</SubmitButton>
      </div>
    </form>
  )
}
