import { useState } from 'react'
import { getErrorMessage } from '@employee360/api-client'
import { Card, EmptyState, HolidayCalendar, InlineAlert, PageContainer, PageHeader } from '@employee360/ui'
import { useHolidayCalendarSettingsQuery, useHolidayCategoriesQuery, useHolidaysQuery } from '../queries/holidayQueries'
import { YearSelector } from '../components/YearSelector'
import { useUiStore, type HolidayViewMode } from '../../../stores/uiStore'
import { ViewToggle } from '../components/ViewToggle'
import { HolidayDetailsDialog } from '../components/HolidayDetailsDialog'
import { HolidayList } from '../components/HolidayList'
import { HolidaysSkeleton } from '../components/HolidaysSkeleton'

const secondaryBtn =
  'h-9 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'

export default function HolidaysPage() {
  const [year, setYear] = useState(() => new Date().getFullYear())
  // 1-based month shown in the calendar; month navigation is clamped to the selected year.
  const [month, setMonth] = useState(() => new Date().getMonth() + 1)
  const changeYear = (y: number) => {
    setYear(y)
    const now = new Date()
    setMonth(now.getFullYear() === y ? now.getMonth() + 1 : 1)
  }
  const [selectedId, setSelectedId] = useState<string>()
  const scope = useUiStore((s) => s.holidayScope)
  const setScope = useUiStore((s) => s.setHolidayScope)
  const storeView = useUiStore((s) => s.holidayView)
  const setStoreView = useUiStore((s) => s.setHolidayView)
  // Tenant default view; local so applying it never writes to the shared store during render.
  const [viewOverride, setViewOverride] = useState<HolidayViewMode>()
  const view = viewOverride ?? storeView
  const changeView = (v: HolidayViewMode) => {
    setViewOverride(v)
    setStoreView(v)
  }
  const settings = useHolidayCalendarSettingsQuery().data
  const weekStartDay = settings?.week_start_day ?? 0
  // Apply the tenant's default view/year once per settings change (adjusting state during render,
  // keyed on a settings stamp), so manual changes stick afterwards.
  const settingsStamp = settings ? `${settings.updated_at}|${settings.default_view}|${settings.default_year}` : undefined
  const [appliedStamp, setAppliedStamp] = useState<string>()
  if (settings && settingsStamp !== appliedStamp) {
    setAppliedStamp(settingsStamp)
    if (settings.default_view === 'calendar' || settings.default_view === 'list') setViewOverride(settings.default_view)
    if (settings.default_year) changeYear(settings.default_year)
  }
  const [categoryId, setCategoryId] = useState('')
  const holidaysQuery = useHolidaysQuery(year, categoryId)
  const categoriesQuery = useHolidayCategoriesQuery()
  const categoryStatus = categoriesQuery.isPending ? 'loading' : categoriesQuery.isError ? 'error' : 'ready'
  const categories = (categoriesQuery.data ?? []).filter((c) => c.is_active)
  const holidays = holidaysQuery.data

  const selected = selectedId ? holidays?.find((h) => h.id === selectedId) : undefined

  return (
    <PageContainer>
      <PageHeader title="Holidays" description="Your organisation's holidays for the year." />
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <YearSelector year={year} onChange={changeYear} />
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-4">
          <div className="flex items-center gap-2">
            <label htmlFor="holiday-category" className="text-sm font-medium text-slate-900">
              Category
            </label>
            <select
              id="holiday-category"
              value={categoryId}
              disabled={categoryStatus !== 'ready'}
              onChange={(e) => setCategoryId(e.target.value)}
              className="h-9 min-w-0 flex-1 rounded-sm border border-slate-300 bg-white px-3 text-sm text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:flex-none"
            >
              <option value="">All categories</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>
          <ViewToggle view={view} onChange={changeView} />
        </div>
      </div>
      {categoriesQuery.isError && (
        <InlineAlert tone="error">Categories could not be loaded, so the category filter is unavailable.</InlineAlert>
      )}
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
        {holidays && view === 'calendar' && (
          <HolidayCalendar
            cursor={{ year, month }}
            onCursorChange={(c) => {
              setYear(c.year)
              setMonth(c.month)
            }}
            minMonth={{ year, month: 1 }}
            maxMonth={{ year, month: 12 }}
            scope={scope}
            onScopeChange={setScope}
            holidays={holidays}
            weekStartDay={weekStartDay}
            onSelect={(h) => setSelectedId(h.id)}
          />
        )}
        {holidays && view === 'list' && holidays.length === 0 && (
          <EmptyState
            title={categoryId ? `No holidays in this category for ${year}.` : `No holidays have been published for ${year} yet.`}
          />
        )}
        {holidays && view === 'list' && holidays.length > 0 && <HolidayList holidays={holidays} categoryStatus={categoryStatus} />}
      </Card>
      {selected && <HolidayDetailsDialog holiday={selected} categoryStatus={categoryStatus} onClose={() => setSelectedId(undefined)} />}
    </PageContainer>
  )
}
