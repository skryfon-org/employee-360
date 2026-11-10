import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  deleteApiV1PositionsId,
  getApiV1Positions,
  postApiV1Positions,
  putApiV1PositionsId,
  unwrapListResponse,
  unwrapSingleEntity,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesPositionCreatePositionRequest as CreatePositionRequest,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesPositionPositionResponse as Position,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesPositionUpdatePositionRequest as UpdatePositionRequest,
} from '@employee360/api-client'
import type { PositionListParams } from '../schemas/positionSchemas'

export type { Position }

export const POSITIONS_KEY = ['positions'] as const

export function usePositionsQuery(params: PositionListParams) {
  return useQuery({
    queryKey: [...POSITIONS_KEY, params],
    queryFn: async ({ signal }) => unwrapListResponse(await getApiV1Positions(params, signal)),
    placeholderData: keepPreviousData,
  })
}

export const ACTIVE_POSITIONS_KEY = [...POSITIONS_KEY, 'active-options'] as const

/** Active positions for selects (single page of up to 100). */
export function useActivePositionsQuery() {
  return useQuery({
    queryKey: ACTIVE_POSITIONS_KEY,
    queryFn: async ({ signal }) =>
      unwrapListResponse(await getApiV1Positions({ is_active: true, page: 1, page_size: 100 }, signal)).data,
  })
}

function useInvalidatingMutation<TVars, TResult>(fn: (vars: TVars) => Promise<TResult>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => qc.invalidateQueries({ queryKey: POSITIONS_KEY }),
  })
}

export const useCreatePositionMutation = () =>
  useInvalidatingMutation(async (body: CreatePositionRequest) => unwrapSingleEntity(await postApiV1Positions(body)))

export const useUpdatePositionMutation = () =>
  useInvalidatingMutation(async (v: { id: string; body: UpdatePositionRequest }) =>
    unwrapSingleEntity(await putApiV1PositionsId(v.id, v.body)),
  )

export const useDeletePositionMutation = () =>
  useInvalidatingMutation(async (id: string) => {
    await deleteApiV1PositionsId(id)
  })
