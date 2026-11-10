import { useState } from 'react'
import { Link } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { Card, EmptyState, InlineAlert, PageContainer, PageHeader } from '@employee360/ui'
import {
  useDeleteHolidayMutation,
  useHolidayCalendarSettingsQuery,
  useHolidayCategoriesQuery,
  useHolidaysQuery,
  type Holiday,
} from '../queries/holidayQueries'
import { useHolidayViewStore } from '../../../stores/holidayViewStore'
import { useToast } from '../../../hooks/useToast'
import { toIsoDate } from '../dates'
import { YearSelector } from '../components/YearSelector'
import { ViewToggle } from '../components/ViewToggle'
import { HolidayCalendar, type CalendarCursor } from '../components/HolidayCalendar'
import { HolidayList } from '../components/HolidayList'
import { CategoryLegend } from '../components/CategoryLegend'
import { HolidaysSkeleton } from '../components/HolidaysSkeleton'
import { HolidayFormModal } from '../components/HolidayFormModal'
import { ConfirmDeleteHolidayModal } from '../components/ConfirmDeleteHolidayModal'

type Dialog = { kind: 'create'; date?: string } | { kind: 'edit'; holiday: Holiday } | { kind: 'delete'; holiday: Holiday } | null

const primaryBtn =
  'inline-flex h-11 items-center rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9'
const secondaryBtn =
  'h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9'

export default function HolidaysPage() {
  const toast = useToast()
  const [cursor, setCursor] = useState<CalendarCursor>(() => {
    const now = new Date()
    return { year: now.getFullYear(), month: now.getMonth() + 1 }
  })
  const year = cursor.year
  const setYear = (y: number) => setCursor((c) => ({ ...c, year: y }))
  const [dialog, setDialog] = useState<Dialog>(null)
  const viewMode = useHolidayViewStore((s) => s.viewMode)
  const setViewMode = useHolidayViewStore((s) => s.setViewMode)
  const settings = useHolidayCalendarSettingsQuery().data
  const weekStartDay = settings?.week_start_day ?? 0
  // Apply the tenant's default view/year whenever the settings change (first load, or after an admin edits
  // them). Adjusting state during render is React's sanctioned alternative to a setState-in-effect.
  const [appliedSettings, setAppliedSettings] = useState<string>()
  if (settings) {
    const stamp = `${settings.updated_at}|${settings.default_view}|${settings.default_year}`
    if (appliedSettings !== stamp) {
      setAppliedSettings(stamp)
      if (settings.default_view === 'calendar' || settings.default_view === 'list') setViewMode(settings.default_view)
      if (settings.default_year) setYear(settings.default_year)
    }
  }
  const holidaysQuery = useHolidaysQuery(year)
  const categoriesQuery = useHolidayCategoriesQuery()
  const del = useDeleteHolidayMutation()
  const [deleteError, setDeleteError] = useState<string>()
  const categories = categoriesQuery.data ?? []
  const activeCategories = categories.filter((c) => c.is_active)
  const holidays = holidaysQuery.data

  const closeDialog = () => {
    setDialog(null)
    setDeleteError(undefined)
  }

  const confirmDelete = (h: Holiday) =>
    del.mutate(h.id, {
      onSuccess: () => {
        toast.success(`Holiday "${h.name}" deleted.`)
        closeDialog()
      },
      onError: (err) => setDeleteError(getErrorMessage(err, 'Could not delete holiday.')),
    })

  return (
    <PageContainer>
      <PageHeader
        title="Holidays"
        description="Maintain the yearly holiday calendar for your organisation."
        actions={
          <>
            <Link to="/holidays/categories" className={secondaryBtn + ' inline-flex items-center'}>
              Categories
            </Link>
            <Link to="/holidays/settings" className={secondaryBtn + ' inline-flex items-center'}>
              Settings
            </Link>
            <button type="button" className={primaryBtn} onClick={() => setDialog({ kind: 'create' })}>
              Add holiday
            </button>
          </>
        }
      />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <YearSelector year={year} onChange={setYear} />
        <ViewToggle />
      </div>
      <CategoryLegend categories={categories} />
      <Card padded={false} role="region" aria-label="Holidays" className="overflow-hidden">
        {holidaysQuery.isPending && <HolidaysSkeleton />}
        {holidaysQuery.isError && (
          <div className="flex flex-col items-start gap-3 p-4">
            <InlineAlert tone="error">{getErrorMessage(holidaysQuery.error, 'Could not load holidays.')}</InlineAlert>
            <button type="button" className={secondaryBtn} onClick={() => void holidaysQuery.refetch()}>
              Retry
            </button>
          </div>
        )}
        {holidays && holidays.length === 0 && (
          <EmptyState
            title={`No holidays have been added for ${year} yet.`}
            action={
              <button type="button" className={`${secondaryBtn} mt-2`} onClick={() => setDialog({ kind: 'create' })}>
                Add the first holiday
              </button>
            }
          />
        )}
        {holidays && holidays.length > 0 && viewMode === 'calendar' && (
          <HolidayCalendar
            cursor={cursor}
            weekStartDay={weekStartDay}
            holidays={holidays}
            onCursorChange={setCursor}
            onSelect={(holiday) => setDialog({ kind: 'edit', holiday })}
            onDateClick={(date) => setDialog({ kind: 'create', date })}
          />
        )}
        {holidays && holidays.length > 0 && viewMode === 'list' && (
          <HolidayList
            holidays={holidays}
            onEdit={(holiday) => setDialog({ kind: 'edit', holiday })}
            onDelete={(holiday) => setDialog({ kind: 'delete', holiday })}
          />
        )}
      </Card>
      {(dialog?.kind === 'create' || dialog?.kind === 'edit') && (
        <HolidayFormModal
          holiday={dialog.kind === 'edit' ? dialog.holiday : undefined}
          categories={activeCategories}
          categoriesLoading={categoriesQuery.isPending}
          categoriesError={categoriesQuery.isError ? getErrorMessage(categoriesQuery.error, 'Could not load categories.') : undefined}
          defaultDate={dialog.kind === 'create' && dialog.date ? dialog.date : toIsoDate(year, cursor.month, 1)}
          onClose={closeDialog}
        />
      )}
      {dialog?.kind === 'delete' && (
        <ConfirmDeleteHolidayModal
          name={dialog.holiday.name}
          loading={del.isPending}
          error={deleteError}
          onConfirm={() => confirmDelete(dialog.holiday)}
          onCancel={closeDialog}
        />
      )}
    </PageContainer>
  )
}
