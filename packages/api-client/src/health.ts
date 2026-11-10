import { getApiV1Health, useGetApiV1Health } from './generated/hooks/health/health.ts';
import type { InternalDeliveryHttpHandlersHealthResponse } from './generated/models/internalDeliveryHttpHandlersHealthResponse.ts';

/**
 * The `data` payload of `GET /api/v1/health`, unwrapped from the response
 * envelope and narrowed to guarantee all three fields are present strings
 * (the generated model marks them optional since Swagger can't express
 * "always present" for a plain struct).
 */
export interface HealthResponse {
  status: string;
  app: string;
  database: string;
}

function assertHealthResponse(
  data: InternalDeliveryHttpHandlersHealthResponse | undefined,
): HealthResponse {
  if (
    !data ||
    typeof data.status !== 'string' ||
    typeof data.app !== 'string' ||
    typeof data.database !== 'string'
  ) {
    throw new Error('Health endpoint responded with an unexpected payload shape');
  }
  return { status: data.status, app: data.app, database: data.database };
}

/**
 * Calls `GET /api/v1/health` via the shared Axios client and returns the
 * typed, unwrapped `HealthResponse`. Built on top of the Orval-generated
 * `getApiV1Health` (see `generated/hooks/health/health.ts`), which already
 * routes through `client.ts`'s `apiRequest` mutator.
 *
 * Named `fetchHealth` (not `getHealth`) to avoid colliding with the
 * Orval-generated `getHealth`/`getHealthz` functions for the unversioned
 * `/health` and `/healthz` routes, which this package also re-exports.
 */
export async function fetchHealth(signal?: AbortSignal): Promise<HealthResponse> {
  const envelope = await getApiV1Health(signal);
  return assertHealthResponse(envelope.data);
}

/**
 * TanStack Query hook for the `/api/v1/health` smoke-test endpoint. Thin
 * wrapper over the Orval-generated `useGetApiV1Health`, narrowed via
 * `select` to the unwrapped `HealthResponse` shape so consumers never touch
 * the raw envelope.
 */
export function useHealth() {
  return useGetApiV1Health({
    query: {
      select: (envelope) => assertHealthResponse(envelope.data),
    },
  });
}
