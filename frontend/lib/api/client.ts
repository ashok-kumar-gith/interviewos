/**
 * Typed fetch wrapper for the InterviewOS backend.
 *
 * - Targets `${NEXT_PUBLIC_API_BASE_URL}/api/v1`.
 * - Attaches a Bearer access token via a pluggable token provider (the access
 *   token is held in memory per ADR §5.3 — never in localStorage).
 * - Parses the standard success/error envelope from ARCHITECTURE §13:
 *     success: { "data": ..., "meta": ... }
 *     error:   { "error": { code, message, details?, request_id } }
 */

const API_ROOT = (process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080").replace(
  /\/$/,
  "",
);
const API_BASE = `${API_ROOT}/api/v1`;

/** Canonical error codes from pkg/apierror (ARCHITECTURE §13). */
export type ApiErrorCode =
  | "BAD_REQUEST"
  | "UNAUTHENTICATED"
  | "INVALID_CREDENTIALS"
  | "REFRESH_TOKEN_INVALID"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "VALIDATION_ERROR"
  | "RATE_LIMITED"
  | "AI_UNAVAILABLE"
  | "INTERNAL";

export interface ApiErrorDetail {
  field: string;
  message: string;
}

export interface ApiErrorEnvelope {
  error: {
    code: ApiErrorCode | string;
    message: string;
    details?: ApiErrorDetail[];
    request_id?: string;
  };
}

export interface ApiMeta {
  total?: number;
  page?: number;
  page_size?: number;
  [key: string]: unknown;
}

export interface ApiSuccessEnvelope<T> {
  data: T;
  meta?: ApiMeta;
}

/** Thrown for any non-2xx response or transport failure. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: ApiErrorCode | string;
  readonly details?: ApiErrorDetail[];
  readonly requestId?: string;

  constructor(
    status: number,
    code: ApiErrorCode | string,
    message: string,
    details?: ApiErrorDetail[],
    requestId?: string,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
    this.requestId = requestId;
  }
}

/* --------------------------------------------------------------------------
 * Auth token hook (access token in memory only).
 * ------------------------------------------------------------------------ */

type TokenProvider = () => string | null | undefined;

let accessTokenProvider: TokenProvider = () => null;

/** Register how the client obtains the current access token. */
export function setAccessTokenProvider(provider: TokenProvider): void {
  accessTokenProvider = provider;
}

/* --------------------------------------------------------------------------
 * Transparent access-token refresh on 401.
 *
 * The access token lives in memory only (ADR §5.3) and is short-lived, so it is
 * absent after a reload and expires mid-session. When a request comes back 401
 * we call a registered refresh handler (which mints a fresh access token from
 * the HttpOnly refresh cookie) exactly once, then retry the original request.
 * Concurrent 401s share a single in-flight refresh so we never stampede.
 * ------------------------------------------------------------------------ */

type RefreshHandler = () => Promise<boolean>;

let refreshHandler: RefreshHandler | null = null;
let inflightRefresh: Promise<boolean> | null = null;

/**
 * Register how the client refreshes the access token. The handler should mint a
 * new access token (via POST /auth/refresh, which sends the HttpOnly cookie),
 * store it, and resolve true on success or false when the session is truly gone.
 */
export function setRefreshHandler(handler: RefreshHandler): void {
  refreshHandler = handler;
}

/** Run the refresh handler, coalescing concurrent callers onto one attempt. */
function refreshAccessToken(): Promise<boolean> {
  if (!refreshHandler) return Promise.resolve(false);
  if (!inflightRefresh) {
    inflightRefresh = refreshHandler()
      .catch(() => false)
      .finally(() => {
        inflightRefresh = null;
      });
  }
  return inflightRefresh;
}

export interface RequestOptions extends Omit<RequestInit, "body"> {
  /** JSON-serializable body; set automatically with the correct content-type. */
  body?: unknown;
  /** Query params appended to the path. */
  query?: Record<string, string | number | boolean | undefined | null>;
}

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  const url = new URL(`${API_BASE}${path.startsWith("/") ? path : `/${path}`}`);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== null) url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

/**
 * Performs the raw fetch with the Accept/Authorization/Content-Type headers and
 * the refresh cookie. On a 401 it transparently refreshes the access token once
 * and replays the request, so callers never see spurious auth failures from an
 * expired/absent in-memory token. `path` requests skip the retry only when
 * `allowRetry` is false (used internally to avoid recursing on /auth/refresh).
 */
async function performRequest(
  path: string,
  options: RequestOptions,
  allowRetry: boolean,
): Promise<Response> {
  const { body, query, headers, ...rest } = options;

  const send = (): Promise<Response> => {
    const finalHeaders = new Headers(headers);
    finalHeaders.set("Accept", "application/json");

    const token = accessTokenProvider();
    if (token) finalHeaders.set("Authorization", `Bearer ${token}`);

    let serializedBody: BodyInit | undefined;
    if (body !== undefined) {
      finalHeaders.set("Content-Type", "application/json");
      serializedBody = JSON.stringify(body);
    }

    return fetch(buildUrl(path, query), {
      ...rest,
      headers: finalHeaders,
      body: serializedBody,
      credentials: "include", // send the HttpOnly refresh cookie when present
    });
  };

  let res: Response;
  try {
    res = await send();
  } catch {
    throw new ApiError(
      0,
      "INTERNAL",
      "Network request failed. Check your connection and retry.",
    );
  }

  // Transparent re-auth: on 401, refresh the access token once and replay.
  if (res.status === 401 && allowRetry && !isAuthPath(path)) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      try {
        res = await send();
      } catch {
        throw new ApiError(
          0,
          "INTERNAL",
          "Network request failed. Check your connection and retry.",
        );
      }
    }
  }

  return res;
}

/** Auth endpoints must not trigger the 401→refresh→retry loop. */
function isAuthPath(path: string): boolean {
  return path.startsWith("/auth/");
}

/**
 * Core request method. Returns the unwrapped `data` from the success envelope.
 * Throws `ApiError` on any non-2xx response.
 */
export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const res = await performRequest(path, options, true);

  if (res.status === 204) {
    return undefined as T;
  }

  const text = await res.text();
  const payload: unknown = text ? safeJsonParse(text) : undefined;

  if (!res.ok) {
    const envelope = payload as ApiErrorEnvelope | undefined;
    const err = envelope?.error;
    throw new ApiError(
      res.status,
      err?.code ?? "INTERNAL",
      err?.message ?? res.statusText ?? "Request failed.",
      err?.details,
      err?.request_id,
    );
  }

  // The backend wraps LIST responses in a { data, meta } envelope but returns
  // single-object responses (auth tokens, dashboard, today, profile, …) flat at
  // the top level. Unwrap `.data` only when the payload is actually an envelope;
  // otherwise return it as-is. (Unwrapping unconditionally returned undefined for
  // every flat response, which broke login + every single-object page.)
  return unwrap<T>(payload);
}

/**
 * Returns the `data` field when `payload` is a success envelope ({ data, meta }),
 * otherwise returns the payload itself. An envelope is recognized as an object
 * whose only keys are `data` and (optionally) `meta`, so domain objects that
 * happen to contain a `data` field are not misread.
 */
function unwrap<T>(payload: unknown): T {
  if (payload && typeof payload === "object" && "data" in payload) {
    const keys = Object.keys(payload as Record<string, unknown>);
    const isEnvelope = keys.every((k) => k === "data" || k === "meta");
    if (isEnvelope) return (payload as ApiSuccessEnvelope<T>).data;
  }
  return payload as T;
}

function safeJsonParse(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return undefined;
  }
}

/**
 * Like {@link apiFetch} but preserves the pagination envelope ({ data, meta })
 * for list endpoints. Returns `{ data, meta }` with `meta` defaulted when the
 * backend omits it.
 */
export async function apiFetchList<T>(
  path: string,
  options: RequestOptions = {},
): Promise<{ data: T[]; meta: ApiMeta }> {
  const res = await performRequest(path, options, true);

  const text = await res.text();
  const payload: unknown = text ? safeJsonParse(text) : undefined;

  if (!res.ok) {
    const envelope = payload as ApiErrorEnvelope | undefined;
    const err = envelope?.error;
    throw new ApiError(
      res.status,
      err?.code ?? "INTERNAL",
      err?.message ?? res.statusText ?? "Request failed.",
      err?.details,
      err?.request_id,
    );
  }

  const env = payload as ApiSuccessEnvelope<T[]> | undefined;
  const data = env?.data ?? [];
  return { data, meta: env?.meta ?? {} };
}

export const api = {
  get: <T>(path: string, options?: RequestOptions) =>
    apiFetch<T>(path, { ...options, method: "GET" }),
  getList: <T>(path: string, options?: RequestOptions) =>
    apiFetchList<T>(path, { ...options, method: "GET" }),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    apiFetch<T>(path, { ...options, method: "POST", body }),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    apiFetch<T>(path, { ...options, method: "PUT", body }),
  patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    apiFetch<T>(path, { ...options, method: "PATCH", body }),
  delete: <T>(path: string, options?: RequestOptions) =>
    apiFetch<T>(path, { ...options, method: "DELETE" }),
};

export { API_BASE };
