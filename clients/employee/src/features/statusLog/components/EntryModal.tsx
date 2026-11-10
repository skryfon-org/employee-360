import { useCallback, useRef, useState } from 'react'
import { Modal } from '../../../components/ui/Modal'
import { EntryForm } from './EntryForm'

type FormProps = Parameters<typeof EntryForm>[0]

/**
 * Add/edit dialog that asks before discarding unsaved changes. `onClose` closes immediately
 * (use it after a successful save); Escape, backdrop and Cancel go through the dirty check.
 */
export function EntryModal({ title, onClose, ...form }: { title: string; onClose: () => void } & Omit<FormProps, 'onCancel' | 'onDirtyChange'>) {
  const dirty = useRef(false)
  const [confirming, setConfirming] = useState(false)
  const onDirtyChange = useCallback((d: boolean) => {
    dirty.current = d
  }, [])
  const requestClose = () => {
    if (dirty.current) setConfirming(true)
    else onClose()
  }

  return (
    <>
      <Modal title={title} onClose={requestClose}>
        <EntryForm {...form} onCancel={requestClose} onDirtyChange={onDirtyChange} />
      </Modal>
      {confirming && (
        <Modal title="Discard changes?" onClose={() => setConfirming(false)}>
          <p className="mb-4 text-sm text-slate-700">You have unsaved changes. If you discard them, they will be lost.</p>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setConfirming(false)}
              className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
            >
              Keep editing
            </button>
            <button
              type="button"
              onClick={onClose}
              className="h-11 rounded-sm bg-red-600 px-4 text-sm font-semibold text-white hover:bg-red-700 active:bg-red-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-600 focus-visible:ring-offset-2 md:h-9"
            >
              Discard
            </button>
          </div>
        </Modal>
      )}
    </>
  )
}
