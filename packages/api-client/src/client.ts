import axios, { type AxiosRequestConfig, type AxiosResponse } from 'axios';
import {
  applyExternalSession,
  clearSession,
  getSession,
  readPersistedSession,
  setSession,
} from './session.ts';

/**
 * Configuration accepted by {@link configureApiClient}.
 *
 * This package is framework-independent: it never reaches into
 * `import.meta.env` or `process.env` itself. Each consuming Vite app is
 * responsible for reading its own environment (e.g.
 * `import.meta.env.VITE_API_BASE_URL`) and passing it in explicitly.
 */
export interface ApiClientConfig {
  baseURL: string;
}

/**
 * Sensible fallback so the client is usable (e.g. in tests) even if
 * `configureApiClient` is never called.
 */
const DEFAULT_BASE_URL = 'http://localhost:8080';

/**
 * The shared Axios instance used by every generated hook and hand-written
 * wrapper in this package. Feature code should never construct its own
 * Axios instance — import this one (or go through the generated hooks,
 * which are wired to it via the mutator below).
 */
export const apiClient = axios.create({
  baseURL: DEFAULT_BASE_URL,
});

/**
 * Configure the shared Axios instance. Call this once from each consuming
 * app's entrypoint (e.g. `main.tsx`), before rendering, passing that app's
 * own environment configuration:
 *
 *   configureApiClient({ baseURL: import.meta.env.VITE_API_BASE_URL })
 */
export function configureApiClient(config: ApiClientConfig): void {
  if (config.baseURL) {
    apiClient.defaults.baseURL = config.baseURL;
  }
}

// --- Auth: request + response interceptors -------------------------------

declare module 'axios' {
  interface AxiosRequestConfig {
    /** Internal: set once a request has been replayed after a silent refresh. */
    _retried?: boolean;
  }
}

export interface AuthConfig {
  /**
   * Called when the session can no longer be recovered (refresh failed or no
   * refresh token). The session has already been cleared; the app should
   * redirect to its login page.
   */
  onAuthFailure?: () => void;
}

let authConfig: AuthConfig = {};

export function configureAuth(config: AuthConfig): void {
  authConfig = { ...authConfig, ...config };
}

const REFRESH_URL = '/api/v1/auth/refresh';
// Endpoints whose 401 means "bad credentials/token", never "expired access token".
const NO_REFRESH_PATHS = [
  '/api/v1/auth/login',
  '/api/v1/auth/refresh',
  '/api/v1/auth/logout',
  '/api/v1/auth/forgot-password',
  '/api/v1/auth/reset-password',
  '/api/v1/invitations/accept',
];

function isNoRefreshUrl(url: string | undefined): boolean {
  if (!url) return false;
  return NO_REFRESH_PATHS.some((p) => url.split('?')[0].endsWith(p));
}

apiClient.interceptors.request.use((config) => {
  const session = getSession();
  if (session?.accessToken && !config.headers.has('Authorization')) {
    config.headers.set('Authorization', `Bearer ${session.accessToken}`);
  }
  // Note: the backend derives tenant from JWT claims and ignores this header
  // (Invariant 1); it is sent for observability/CORS-allowed parity only.
  if (session?.tenantId && !config.headers.has('X-Tenant-ID')) {
    config.headers.set('X-Tenant-ID', session.tenantId);
  }
  return config;
});

interface RefreshEnvelope {
  data?: { access_token?: string; refresh_token?: string; expires_at?: string };
}

/** The session is unrecoverable (no/revoked refresh token): log the user out. */
class SessionExpiredError extends Error {
  constructor(message = 'No refresh token available') {
    super(message);
    this.name = 'SessionExpiredError';
  }
}

/** Abort a hung refresh so the cross-tab Web Lock is never held indefinitely. */
const REFRESH_TIMEOUT_MS = 15_000;

/**
 * Only an explicit auth rejection from the refresh endpoint (400/401/403)
 * means the session is dead. Network errors, timeouts, 5xx and 429 are
 * transient and must not log the user out.
 */
function isAuthRejection(err: unknown): boolean {
  if (err instanceof SessionExpiredError) return true;
  const status = axios.isAxiosError(err) ? err.response?.status : undefined;
  return status === 400 || status === 401 || status === 403;
}

let refreshPromise: Promise<string> | null = null;

const REFRESH_LOCK_NAME = 'employee360-auth-refresh';

/**
 * Run `fn` under a cross-tab exclusive Web Lock so two tabs never present the
 * same single-use refresh token. Runs `fn` directly where Web Locks are
 * unavailable (old browsers, SSR, tests).
 */
async function withRefreshLock<T>(fn: () => Promise<T>): Promise<T> {
  const locks = typeof navigator !== 'undefined' ? navigator.locks : undefined;
  if (!locks || typeof locks.request !== 'function') return fn();
  return await locks.request<Promise<T>>(REFRESH_LOCK_NAME, fn);
}

/**
 * Rotate the token pair. Concurrent callers in a tab share one in-flight
 * promise; across tabs a Web Lock serialises refreshes, and inside the lock
 * the persisted session is re-read: if another tab already rotated the
 * refresh token, its access token is reused instead of calling the backend
 * (the backend revokes the whole token family on refresh-token reuse).
 * Rejects without clearing the session; callers decide whether that means
 * logout.
 */
export function refreshSession(): Promise<string> {
  if (refreshPromise) return refreshPromise;
  const session = getSession();
  if (!session?.refreshToken) {
    return Promise.reject(new SessionExpiredError());
  }
  const staleRefreshToken = session.refreshToken;
  refreshPromise = withRefreshLock(async () => {
    const latest = readPersistedSession();
    if (!latest?.refreshToken) {
      // Logged out (or storage cleared) in another tab while we waited.
      throw new SessionExpiredError();
    }
    if (latest.refreshToken !== staleRefreshToken) {
      // Another tab rotated already: adopt its session, skip the backend.
      if (getSession()?.refreshToken !== latest.refreshToken) applyExternalSession(latest);
      return latest.accessToken;
    }
    const { data } = await apiClient.post<RefreshEnvelope>(
      REFRESH_URL,
      { refresh_token: latest.refreshToken },
      { timeout: REFRESH_TIMEOUT_MS },
    );
    // Logged out (this tab or another) while the request was in flight: do
    // not resurrect the session by writing the freshly rotated tokens.
    if (!getSession() || !readPersistedSession()) throw new SessionExpiredError();
    const tokens = data.data;
    if (!tokens?.access_token || !tokens.refresh_token) {
      throw new Error('Refresh endpoint returned an unexpected payload');
    }
    setSession({
      accessToken: tokens.access_token,
      refreshToken: tokens.refresh_token,
      tenantId: latest.tenantId,
      expiresAt: tokens.expires_at,
    });
    return tokens.access_token;
  }).finally(() => {
    refreshPromise = null;
  });
  return refreshPromise;
}

function forceLogout(): void {
  clearSession();
  authConfig.onAuthFailure?.();
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error: unknown) => {
    if (!axios.isAxiosError(error) || !error.config) return Promise.reject(error);
    const original = error.config;
    if (
      error.response?.status !== 401 ||
      original._retried ||
      isNoRefreshUrl(original.url)
    ) {
      return Promise.reject(error);
    }
    // No session at all (e.g. anonymous request): nothing to refresh or log out.
    if (!getSession()) return Promise.reject(error);

    // The request may have been sent with a token that has since been rotated
    // (here or in another tab). Replay with the newest one without refreshing.
    const sent = /^Bearer (.+)$/.exec(String(original.headers?.get?.('Authorization') ?? ''))?.[1];
    const newest = getSession()?.accessToken;
    let accessToken: string;
    if (sent && newest && sent !== newest) {
      accessToken = newest;
    } else {
      try {
        accessToken = await refreshSession();
      } catch (refreshError) {
        if (isAuthRejection(refreshError)) {
          forceLogout();
          return Promise.reject(error);
        }
        // Transient (offline, timeout, 5xx): keep the session, surface the failure.
        return Promise.reject(refreshError);
      }
    }
    original._retried = true; // exactly one replay; a second 401 is surfaced as-is
    original.headers.set('Authorization', `Bearer ${accessToken}`);
    return apiClient.request(original);
  },
);

/**
 * The Orval custom mutator (see `orval.config.ts` -> `override.mutator`).
 * Every Orval-generated hook/function calls this instead of raw axios, so
 * generated code always goes through the shared instance above (and its
 * interceptors) rather than creating its own.
 */
export const apiRequest = async <T>(config: AxiosRequestConfig): Promise<T> => {
  const response: AxiosResponse<T> = await apiClient.request<T>(config);
  return response.data;
};

export default apiClient;
