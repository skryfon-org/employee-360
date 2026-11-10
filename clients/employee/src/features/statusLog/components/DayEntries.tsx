import { Card, InlineAlert, useToast } from '@employee360/ui'
import { useEffect, useMemo, useState } from 'react'
import { Modal } from '../../../components/ui/Modal'
import { EntryModal } from './EntryModal'
import { projectsActiveOn } from '../allocations'
import { isPlainShortcut } from '../shortcuts'
import { ExpandableNotes } from './ExpandableNotes'
import type { ProjectOption } from './EntryForm'
import {
  statusLogErrorMessage,
  useCreateStatusLogMutation,
  useDeleteStatusLogMutation,
  useUpdateStatusLogMutation,
  type StatusLogEntry,
} from '../queries/statusLogQueries'
import { isEditableDate, longDate } from '../dates'
import { DAILY_HOURS_CAP } from '../schemas/statusLogSchemas'
import { entryName } from '../targets'

const btn =
  'h-11 rounded-sm border bg-white px-3 text-sm font-medium focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:text-slate-500 md:h-9'
const primaryBtn =
  'h-11 rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-200 disabled:text-slate-500 md:h-9'

export function DayEntries({ date, entries, total, projects: allProjects }: { date: string; entries: StatusLogEntry[]; total: number; projects: ProjectOption[] }) {
  const [deleting, setDeleting] = useState<StatusLogEntry | null>(null)
  const [modal, setModal] = useState<{ mode: 'add' } | { mode: 'edit'; id: string } | null>(null)
  const create = useCreateStatusLogMutation()
  const update = useUpdateStatusLogMutation()
  const del = useDeleteStatusLogMutation()
  const toast = useToast()

  // Mutation result objects change identity every render; their reset functions are stable.
  const resetCreate = create.reset
  const resetUpdate = update.reset
  const resetDel = del.reset

  const readOnly = !isEditableDate(date)
  const projects = useMemo(() => projectsActiveOn(allProjects, date), [allProjects, date])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (readOnly || (e.key !== 'n' && e.key !== 'N') || !isPlainShortcut(e)) return
      e.preventDefault()
      resetCreate()
      resetUpdate()
      resetDel()
      setModal({ mode: 'add' })
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [resetCreate, resetUpdate, resetDel, readOnly])

  const pct = Math.min(100, (total / DAILY_HOURS_CAP) * 100)

  const editing = modal?.mode === 'edit' ? entries.find((e) => e.id === modal.id) : undefined
  const closeModal = () => {
    create.reset()
    update.reset()
    setModal(null)
  }

  const resetErrors = () => {
    create.reset()
    update.reset()
    del.reset()
  }

  return (
    <Card className="flex flex-col gap-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-col gap-2 sm:min-w-64">
          <h2 className="text-base font-semibold text-slate-900">{longDate(date)}</h2>
          <div className="flex items-center gap-3">
            <div
              role="progressbar"
              aria-label="Hours logged"
              aria-valuemin={0}
              aria-valuemax={DAILY_HOURS_CAP}
              aria-valuenow={total}
              className="h-2 flex-1 overflow-hidden rounded-sm bg-slate-200"
            >
              <div className="h-full bg-slate-900" style={{ width: `${pct}%` }} />
            </div>
            <p className="whitespace-nowrap text-sm text-slate-900" aria-live="polite">
              Total: <strong>{total}h</strong> of {DAILY_HOURS_CAP}h
            </p>
          </div>
        </div>
        {!readOnly && (
        <button
          type="button"
          className={primaryBtn}
          title="Add entry (N)"
          aria-keyshortcuts="N"
          onClick={() => {
            resetErrors()
            setModal({ mode: 'add' })
          }}
        >
          Add entry
        </button>
        )}
      </div>

      {readOnly && (
        <p role="note" className="rounded-sm border border-slate-300 bg-slate-100 px-3 py-2 text-sm font-medium text-slate-900">
          Read-only — older than 7 days
        </p>
      )}

      {del.error != null && <InlineAlert tone="error">{statusLogErrorMessage(del.error)}</InlineAlert>}

      {modal?.mode === 'add' && (
        <EntryModal
          title="New entry"
          onClose={closeModal}
          projects={projects}
          otherHours={total}
          saving={create.isPending}
          error={create.error != null ? statusLogErrorMessage(create.error) : null}
          onSubmit={(body) =>
            create.mutate(
              { data: { ...body, log_date: date } },
              {
                onSuccess: () => {
                  toast.success('Entry added.')
                  closeModal()
                },
              },
            )
          }
        />
      )}

      {editing && (
        <EntryModal
          title="Edit entry"
          onClose={closeModal}
          projects={projects}
          entry={editing}
          otherHours={total - (editing.hours ?? 0)}
          saving={update.isPending}
          error={update.error != null ? statusLogErrorMessage(update.error) : null}
          onSubmit={(body) =>
            update.mutate(
              { id: editing.id ?? '', data: body },
              {
                onSuccess: () => {
                  toast.success('Entry updated.')
                  closeModal()
                },
              },
            )
          }
        />
      )}

      {deleting && (
        <Modal title="Delete entry?" onClose={() => setDeleting(null)}>
          <p className="mb-4 text-sm text-slate-700">
            This will permanently delete <strong>{entryName(deleting)}</strong> ({deleting.hours}h) from {longDate(date)}.
          </p>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setDeleting(null)}
              className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => {
                del.mutate({ id: deleting.id ?? '' }, { onSuccess: () => toast.success('Entry deleted.') })
                setDeleting(null)
              }}
              className="h-11 rounded-sm bg-red-600 px-4 text-sm font-semibold text-white hover:bg-red-700 active:bg-red-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-600 focus-visible:ring-offset-2 md:h-9"
            >
              Delete
            </button>
          </div>
        </Modal>
      )}

      {entries.length === 0 ? (
        <p className="rounded-sm border border-dashed border-slate-300 p-6 text-center text-sm text-slate-600">
          {readOnly ? 'Nothing was logged for this day.' : 'Nothing logged for this day yet.'}
        </p>
      ) : (
        <div className="overflow-x-auto rounded-sm border border-slate-200">
          <table className="w-full min-w-[32rem] table-fixed border-collapse text-left">
            <thead className="bg-slate-100 text-xs font-medium text-slate-600">
              <tr>
                <th scope="col" className="w-[15%] px-4 py-2">Project / category</th>
                <th scope="col" className="w-[12%] px-4 py-2 text-center">Hours</th>
                <th scope="col" className="px-4 py-2">Notes</th>
                <th scope="col" className="w-44 px-4 py-2 text-right">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
                  <tr key={e.id} className="border-t border-slate-200 align-top text-sm text-slate-900 hover:bg-slate-50">
                    <td className="break-words px-4 py-3 font-medium">{entryName(e)}</td>
                    <td className="px-4 py-3 text-center tabular-nums">{e.hours}h</td>
                    <td className="break-words px-4 py-3 text-slate-600"><ExpandableNotes notes={e.notes} /></td>
                    <td className="px-4 py-2">
                      {!readOnly && (
                      <div className="flex justify-end gap-2">
                        <button
                          type="button"
                          aria-label={`Edit ${entryName(e)}`}
                          onClick={() => {
                            resetErrors()
                            setModal({ mode: 'edit', id: e.id ?? '' })
                          }}
                          className={`${btn} border-slate-300 text-slate-800 hover:bg-slate-100 focus-visible:ring-slate-900`}
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          aria-label={`Delete ${entryName(e)}`}
                          disabled={del.isPending}
                          onClick={() => {
                            resetErrors()
                            setDeleting(e)
                          }}
                          className={`${btn} border-red-300 text-red-600 hover:bg-red-50 focus-visible:ring-red-600`}
                        >
                          Delete
                        </button>
                      </div>
                      )}
                    </td>
                  </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  )
}
