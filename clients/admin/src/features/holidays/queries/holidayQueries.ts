import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  deleteApiV1HolidayCategoriesId,
  deleteApiV1HolidaysId,
  getApiV1HolidayCalendarSettings,
  getApiV1HolidayCategories,
  getApiV1Holidays,
  postApiV1HolidayCategories,
  postApiV1Holidays,
  putApiV1HolidayCalendarSettings,
  putApiV1HolidayCategoriesId,
  putApiV1HolidaysId,
  unwrapListResponse,
  unwrapSingleEntity,
  type PageMeta,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesHolidayCategoryResponse as CategoryResponse,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesHolidayCreateCategoryRequest as CreateCategoryRequest,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesHolidayCreateHolidayRequest as CreateHolidayRequest,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesHolidayHolidayResponse as HolidayResponse,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesHolidaySettingsResponse as HolidaySettings,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesHolidayUpdateCategoryRequest as UpdateCategoryRequest,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesHolidayUpdateHolidayRequest as UpdateHolidayRequest,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesHolidayUpdateSettingsRequest as UpdateSettingsRequest,
} from '@employee360/api-client'

/** A category with the fields the UI relies on guaranteed present. */
export interface HolidayCategory {
  id: string
  name: string
  color: string
  description?: string
  is_active: boolean
  is_public: boolean
}

/** A holiday with its category joined in client-side (the API returns `category_id` only). */
export interface Holiday {
  id: string
  name: string
  /** Calendar date, `YYYY-MM-DD`. */
  date: string
  description?: string
  category_id: string
  category?: HolidayCategory
}

export type { HolidaySettings, CreateHolidayRequest, UpdateHolidayRequest }

export const HOLIDAYS_KEY = ['holidays'] as const
export const holidaysYearKey = (year: number) => [...HOLIDAYS_KEY, year] as const
export const HOLIDAY_CATEGORIES_KEY = ['holiday-categories'] as const
export const HOLIDAY_SETTINGS_KEY = ['holiday-calendar-settings'] as const

function toCategory(c: CategoryResponse): HolidayCategory {
  return {
    id: c.id ?? '',
    name: c.name ?? '',
    color: c.color ?? '',
    description: c.description,
    is_active: c.is_active ?? true,
    is_public: c.is_public ?? false,
  }
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
/** Safety cap against a misbehaving server reporting endless pages. */
const MAX_CATEGORY_PAGES = 100

/** Fetches every page of categories, following `meta.total_pages`. */
async function fetchAllCategories(signal?: AbortSignal): Promise<HolidayCategory[]> {
  const all: HolidayCategory[] = []
  for (let page = 1; page <= MAX_CATEGORY_PAGES; page++) {
    const { data, meta } = unwrapListResponse(
      await getApiV1HolidayCategories({ page, page_size: CATEGORY_PAGE_SIZE }, signal),
    )
    all.push(...data.map(toCategory))
    const totalPages = (meta as PageMeta | undefined)?.total_pages
    // Without meta, fall back to "a short page means the last page".
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

/** Holidays for one year, with categories joined from the (cached) categories query. */
export function useHolidaysQuery(year: number) {
  const categories = useHolidayCategoriesQuery().data
  return useQuery({
    queryKey: holidaysYearKey(year),
    queryFn: async ({ signal }) => unwrapListResponse(await getApiV1Holidays({ year }, signal)).data,
    select: (rows): Holiday[] => {
      const byId = new Map((categories ?? []).map((c) => [c.id, c]))
      return rows
        .map((h) => toHoliday(h, byId))
        .sort((a, b) => a.date.localeCompare(b.date) || a.name.localeCompare(b.name))
    },
  })
}

export function useHolidayCalendarSettingsQuery() {
  return useQuery({
    queryKey: HOLIDAY_SETTINGS_KEY,
    queryFn: async ({ signal }) => unwrapSingleEntity(await getApiV1HolidayCalendarSettings(signal)),
  })
}

/** Invalidates every year-keyed holiday query (a date edit can move a holiday between years). */
function useInvalidatingMutation<TVars, TResult>(fn: (vars: TVars) => Promise<TResult>, keys: ReadonlyArray<readonly string[]> = [HOLIDAYS_KEY]) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => Promise.all(keys.map((queryKey) => qc.invalidateQueries({ queryKey }))),
  })
}

export const useCreateHolidayMutation = () =>
  useInvalidatingMutation(async (body: CreateHolidayRequest) => unwrapSingleEntity(await postApiV1Holidays(body)))

export const useUpdateHolidayMutation = () =>
  useInvalidatingMutation(async ({ id, body }: { id: string; body: UpdateHolidayRequest }) =>
    unwrapSingleEntity(await putApiV1HolidaysId(id, body)),
  )

export const useDeleteHolidayMutation = () =>
  useInvalidatingMutation(async (id: string) => {
    await deleteApiV1HolidaysId(id)
  })

const CATEGORY_KEYS = [HOLIDAY_CATEGORIES_KEY, HOLIDAYS_KEY] as const

export const useCreateHolidayCategoryMutation = () =>
  useInvalidatingMutation(
    async (body: CreateCategoryRequest) => unwrapSingleEntity(await postApiV1HolidayCategories(body)),
    CATEGORY_KEYS,
  )

export const useUpdateHolidayCategoryMutation = () =>
  useInvalidatingMutation(
    async ({ id, body }: { id: string; body: UpdateCategoryRequest }) =>
      unwrapSingleEntity(await putApiV1HolidayCategoriesId(id, body)),
    CATEGORY_KEYS,
  )

export const useDeleteHolidayCategoryMutation = () =>
  useInvalidatingMutation(async (id: string) => {
    await deleteApiV1HolidayCategoriesId(id)
  }, CATEGORY_KEYS)

export const useUpdateHolidaySettingsMutation = () =>
  useInvalidatingMutation(
    async (body: UpdateSettingsRequest) => unwrapSingleEntity(await putApiV1HolidayCalendarSettings(body)),
    [HOLIDAY_SETTINGS_KEY],
  )
