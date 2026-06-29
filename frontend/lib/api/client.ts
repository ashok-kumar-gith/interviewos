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
 * Core request method. Returns the unwrapped `data` from the success envelope.
 * Throws `ApiError` on any non-2xx response.
 */
export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, query, headers, ...rest } = options;

  const finalHeaders = new Headers(headers);
  finalHeaders.set("Accept", "application/json");

  const token = accessTokenProvider();
  if (token) finalHeaders.set("Authorization", `Bearer ${token}`);

  let serializedBody: BodyInit | undefined;
  if (body !== undefined) {
    finalHeaders.set("Content-Type", "application/json");
    serializedBody = JSON.stringify(body);
  }

  let res: Response;
  try {
    res = await fetch(buildUrl(path, query), {
      ...rest,
      headers: finalHeaders,
      body: serializedBody,
      credentials: "include", // send the HttpOnly refresh cookie when present
    });
  } catch {
    throw new ApiError(
      0,
      "INTERNAL",
      "Network request failed. Check your connection and retry.",
    );
  }

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

  return (payload as ApiSuccessEnvelope<T> | undefined)?.data as T;
}

function safeJsonParse(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return undefined;
  }
}

export const api = {
  get: <T>(path: string, options?: RequestOptions) =>
    apiFetch<T>(path, { ...options, method: "GET" }),
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
