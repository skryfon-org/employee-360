import axios from 'axios';

/**
 * Extract a user-facing message from an error: the backend's
 * `{ error: { message } }` body for an AxiosError, else `err.message`, else
 * `fallback`.
 */
export function getErrorMessage(err: unknown, fallback = 'Something went wrong'): string {
  if (axios.isAxiosError<{ error?: { message?: string } }>(err)) {
    return err.response?.data?.error?.message || err.message || fallback;
  }
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}

/**
 * Stable error codes returned by the invitation validate/accept endpoints.
 * Features must branch on these, never on message text.
 */
export const INVITATION_ERROR_CODES = {
  INVALID_TOKEN: 'INVALID_TOKEN',
  EXPIRED: 'INVITATION_EXPIRED',
  REVOKED: 'INVITATION_REVOKED',
  ACCEPTED: 'INVITATION_ACCEPTED',
} as const;

/** Error code returned by the invite endpoint when the email domain is not registered for the tenant. */
export const EMAIL_DOMAIN_NOT_ALLOWED = 'EMAIL_DOMAIN_NOT_ALLOWED';

/** Read the backend's `{ error: { code } }` from an AxiosError, else `undefined`. */
export function getErrorCode(err: unknown): string | undefined {
  if (axios.isAxiosError<{ error?: { code?: string } }>(err)) {
    return err.response?.data?.error?.code || undefined;
  }
  // Non-HTTP errors that carry a stable string `code` .
  if (err instanceof Error && typeof (err as unknown as { code?: unknown }).code === 'string') {
    return (err as unknown as { code: string }).code;
  }
  return undefined;
}

/** True for a 409 response whose `error.code` is the generic `CONFLICT` (e.g. duplicate unique value). */
export function isConflictError(err: unknown): boolean {
  return axios.isAxiosError(err) && err.response?.status === 409 && getErrorCode(err) === 'CONFLICT';
}

/** Error codes returned by the super_admin tenant endpoints. Branch on these, never on message text. */
export const TENANT_ERROR_CODES = {
  INVALID_DOMAIN: 'INVALID_DOMAIN',
  INVALID_NAME: 'INVALID_NAME',
  DOMAIN_ALREADY_EXISTS: 'DOMAIN_ALREADY_EXISTS',
  LAST_DOMAIN: 'LAST_DOMAIN',
  DOMAIN_IN_USE: 'DOMAIN_IN_USE',
} as const;
