/** Backend response envelope: `{ success, data?, error?, meta? }`. */
export interface ApiEnvelope<T, M = unknown> {
  success?: boolean;
  data?: T;
  error?: { code?: string; message?: string; details?: unknown };
  meta?: M;
}

/** Returns `data` from a success envelope; throws if failed or payload missing. */
export function unwrapSingleEntity<T>(response: ApiEnvelope<T>): T {
  if (response.success === false || response.data === undefined || response.data === null) {
    throw new Error(response.error?.message ?? 'Request failed: empty response payload');
  }
  return response.data;
}

/** Returns `{ data, meta }` from a list envelope; a missing `data` becomes `[]`. */
export function unwrapListResponse<T, M = unknown>(
  response: ApiEnvelope<T[], M>,
): { data: T[]; meta: M | undefined } {
  if (response.success === false) {
    throw new Error(response.error?.message ?? 'Request failed');
  }
  return { data: response.data ?? [], meta: response.meta };
}
