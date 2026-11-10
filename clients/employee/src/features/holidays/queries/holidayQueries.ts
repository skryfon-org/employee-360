import { useQuery } from '@tanstack/react-query'
import {
  getApiV1HolidayCategories,
  getApiV1HolidayCalendarSettings,
  getApiV1Holidays,
  unwrapSingleEntity,
  unwrapListResponse,
  type PageMeta,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesHolidayCategoryResponse as CategoryResponse,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesHolidayHolidayResponse as HolidayResponse,
} from '@employee360/api-client'

export interface HolidayCategory {
  id: string
  name: string
  color: string
  is_active: boolean
}

export interface Holiday {
  id: string
  name: string
  /** Calendar date, `YYYY-MM-DD`. */
  date: string
  description?: string
  category_id: string
  category?: HolidayCategory
}

export const HOLIDAYS_KEY = ['holidays'] as const
/** Key shape: ['holidays', year, categoryId]; categoryId is '' for "all categories". Prefix-matches HOLIDAYS_KEY. */
export const holidaysYearKey = (year: number, categoryId = '') => [...HOLIDAYS_KEY, year, categoryId] as const
export const HOLIDAY_SETTINGS_KEY = ['holiday-calendar-settings'] as const
export const HOLIDAY_CATEGORIES_KEY = ['holiday-categories'] as const

function toCategory(c: CategoryResponse): HolidayCategory {
  return { id: c.id ?? '', name: c.name ?? '', color: c.color ?? '', is_active: c.is_active ?? true }
}

function toHoliday(h: HolidayResponse, categories: Map<string, HolidayCategory>): Holiday {
  const categoryId = h.category_id ?? ''
  return {
    id: h.id ?? '',
    name: h.name ?? '',
    date: h.date ?? '',
    description: h.description,
    category_id: categoryId,
    category: categories.get(categoryId),
  }
}

const CATEGORY_PAGE_SIZE = 100
const MAX_CATEGORY_PAGES = 100

async function fetchAllCategories(signal?: AbortSignal): Promise<HolidayCategory[]> {
  const all: HolidayCategory[] = []
  for (let page = 1; page <= MAX_CATEGORY_PAGES; page++) {
    const { data, meta } = unwrapListResponse(
      await getApiV1HolidayCategories({ page, page_size: CATEGORY_PAGE_SIZE }, signal),
    )
    all.push(...data.map(toCategory))
    const totalPages = (meta as PageMeta | undefined)?.total_pages
    const done = totalPages !== undefined ? page >= totalPages : data.length < CATEGORY_PAGE_SIZE
    if (done || data.length === 0) break
  }
  return all
}

export function useHolidayCategoriesQuery() {
  return useQuery({
    queryKey: HOLIDAY_CATEGORIES_KEY,
    queryFn: ({ signal }) => fetchAllCategories(signal),
    staleTime: 5 * 60_000,
  })
}

/** Holidays for one year and optional category (both part of the key; filtering is server-side), sorted chronologically, categories joined client-side. */
export function useHolidaysQuery(year: number, categoryId = '') {
  const categories = useHolidayCategoriesQuery().data
  return useQuery({
    queryKey: holidaysYearKey(year, categoryId),
    queryFn: async ({ signal }) =>
      unwrapListResponse(await getApiV1Holidays(categoryId ? { year, category_id: categoryId } : { year }, signal)).data,
    // Keep showing the previous category's rows while switching category (same year only), avoiding a skeleton flash.
    placeholderData: (prev, prevQuery) => (prevQuery?.queryKey[1] === year ? prev : undefined),
    select: (rows): Holiday[] => {
      const byId = new Map((categories ?? []).map((c) => [c.id, c]))
      return rows
        .map((h) => toHoliday(h, byId))
        .sort((a, b) => a.date.localeCompare(b.date) || a.name.localeCompare(b.name))
    },
  })
}

/** Tenant calendar settings (week start, default view/year). Failures are non-blocking: callers fall back to defaults. */
export function useHolidayCalendarSettingsQuery() {
  return useQuery({
    queryKey: HOLIDAY_SETTINGS_KEY,
    queryFn: async ({ signal }) => unwrapSingleEntity(await getApiV1HolidayCalendarSettings(signal)),
    retry: false,
  })
}
