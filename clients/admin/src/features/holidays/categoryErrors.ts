import { getErrorCode } from '@employee360/api-client'

/** Stable backend error code (HTTP 409) for deleting a category that holidays still use. */
export const CATEGORY_IN_USE_CODE = 'CATEGORY_IN_USE'
/** Backend error code (HTTP 409) for a duplicate category name on create/update. */
const DUPLICATE_CODE = 'CONFLICT'

export function isCategoryInUseError(err: unknown): boolean {
  return getErrorCode(err) === CATEGORY_IN_USE_CODE
}

export function isCategoryDuplicateError(err: unknown): boolean {
  return getErrorCode(err) === DUPLICATE_CODE
}
