import { useState } from 'react'
import { Link } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { Card, EmptyState, InlineAlert, PageContainer, PageHeader, RowActionsMenu } from '@employee360/ui'
import {
  useDeleteHolidayCategoryMutation,
  useHolidayCategoriesQuery,
  type HolidayCategory,
} from '../queries/holidayQueries'
import { isCategoryInUseError } from '../categoryErrors'
import { useToast } from '../../../hooks/useToast'
import { CategoryDot } from '../components/CategoryBadge'
import { CategoryFormModal } from '../components/CategoryFormModal'
import { ConfirmDeleteCategoryModal } from '../components/ConfirmDeleteCategoryModal'

export const CATEGORY_IN_USE_MESSAGE =
  'This category is still used by one or more holidays. Move or delete those holidays before deleting the category.'

type Dialog = { kind: 'create' } | { kind: 'edit'; category: HolidayCategory } | { kind: 'delete'; category: HolidayCategory } | null

const primaryBtn =
  'inline-flex h-11 items-center rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9'
const secondaryBtn =
  'inline-flex h-11 items-center md:h-8 rounded-sm border border-slate-300 bg-white px-3 text-xs font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'

const CELL =
  'px-4 py-3 max-sm:flex max-sm:items-start max-sm:justify-between max-sm:gap-4 max-sm:p-0 max-sm:before:shrink-0 max-sm:before:text-xs max-sm:before:font-medium max-sm:before:text-slate-600 max-sm:before:content-[attr(data-label)]'

function StatusBadge({ active }: { active: boolean }) {
  const cls = active ? 'bg-green-50 border-green-300 text-green-700' : 'bg-slate-100 border-slate-300 text-slate-600'
  return (
    <span className={`inline-flex h-6 items-center gap-1.5 rounded-sm border px-2 text-xs font-medium ${cls}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${active ? 'bg-green-700' : 'bg-slate-600'}`} />
      {active ? 'Active' : 'Inactive'}
    </span>
  )
}

export default function HolidayCategoriesPage() {
  const toast = useToast()
  const query = useHolidayCategoriesQuery()
  const del = useDeleteHolidayCategoryMutation()
  const [dialog, setDialog] = useState<Dialog>(null)
  const [deleteError, setDeleteError] = useState<string>()
  const categories = query.data

  const closeDialog = () => {
    setDialog(null)
    setDeleteError(undefined)
  }

  const confirmDelete = (c: HolidayCategory) =>
    del.mutate(c.id, {
      onSuccess: () => {
        toast.success(`Category "${c.name}" deleted.`)
        closeDialog()
      },
      onError: (err) =>
        setDeleteError(isCategoryInUseError(err) ? CATEGORY_IN_USE_MESSAGE : getErrorMessage(err, 'Could not delete category.')),
    })

  return (
    <PageContainer>
      <PageHeader
        title="Holiday categories"
        description="Categories group and colour-code holidays in the calendar."
        actions={
          <>
            <Link to="/holidays" className={secondaryBtn}>
              Back to holidays
            </Link>
            <button type="button" className={primaryBtn} onClick={() => setDialog({ kind: 'create' })}>
              Add category
            </button>
          </>
        }
      />
      <Card padded={false} role="region" aria-label="Categories" className="overflow-hidden">
        {query.isPending && <p className="p-6 text-sm text-slate-600">Loading categories...</p>}
        {query.isError && (
          <div className="flex flex-col items-start gap-3 p-4">
            <InlineAlert tone="error">{getErrorMessage(query.error, 'Could not load categories.')}</InlineAlert>
            <button type="button" className={secondaryBtn} onClick={() => void query.refetch()}>
              Retry
            </button>
          </div>
        )}
        {categories && categories.length === 0 && (
          <EmptyState title="No categories yet" description="Add one before creating holidays." />
        )}
        {categories && categories.length > 0 && (
          <div className="overflow-x-auto">
            <table aria-label="Holiday categories" className="w-full text-left max-sm:block sm:min-w-[40rem]">
              <thead className="bg-slate-100 text-xs font-medium text-slate-600 max-sm:sr-only">
                <tr>
                  <th scope="col" className="px-4 py-3">Category</th>
                  <th scope="col" className="px-4 py-3">Description</th>
                  <th scope="col" className="px-4 py-3">Status</th>
                  <th scope="col" className="px-4 py-3 text-right"><span className="sr-only">Actions</span></th>
                </tr>
              </thead>
              <tbody className="max-sm:block">
                {categories.map((c) => (
                  <tr
                    key={c.id}
                    className="border-t border-slate-200 text-sm text-slate-900 hover:bg-slate-50 max-sm:flex max-sm:flex-col max-sm:gap-2 max-sm:p-4"
                  >
                    <td className="px-4 py-3 max-sm:p-0">
                      <p className="flex items-center gap-2 font-medium">
                        <CategoryDot color={c.color} />
                        <span className="break-words">{c.name}</span>
                        <span className="text-xs font-normal text-slate-600">{c.color}</span>
                        {c.is_public && <span className="rounded-sm border border-slate-300 bg-slate-100 px-1.5 py-0.5 text-xs font-medium text-slate-700">Public</span>}
                      </p>
                    </td>
                    <td className={`${CELL} text-slate-600 sm:max-w-md sm:break-words`} data-label="Description">
                      {c.description || '-'}
                    </td>
                    <td className={CELL} data-label="Status">
                      <StatusBadge active={c.is_active} />
                    </td>
                    <td className="px-4 py-3 max-sm:p-0 max-sm:pt-1">
                      <div className="sm:flex sm:justify-end">
                        <RowActionsMenu
                          label={c.name}
                          actions={[
                            { label: 'Edit', onSelect: () => setDialog({ kind: 'edit', category: c }) },
                            { label: 'Delete', danger: true, onSelect: () => setDialog({ kind: 'delete', category: c }) },
                          ]}
                        />
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
      {(dialog?.kind === 'create' || dialog?.kind === 'edit') && (
        <CategoryFormModal category={dialog.kind === 'edit' ? dialog.category : undefined} onClose={closeDialog} />
      )}
      {dialog?.kind === 'delete' && (
        <ConfirmDeleteCategoryModal
          name={dialog.category.name}
          loading={del.isPending}
          error={deleteError}
          onConfirm={() => confirmDelete(dialog.category)}
          onCancel={closeDialog}
        />
      )}
    </PageContainer>
  )
}
