/**
 * Resume API layer — profile, projects CRUD, and ATS scoring.
 * Shapes mirror the OpenAPI schemas: ResumeProfile, ResumeProfileUpsert,
 * ResumeProject, ResumeProjectUpsert, ResumeScoreResponse.
 */

import { api, API_BASE, ApiError, type ApiErrorEnvelope } from "@/lib/api/client";
import { useAuthStore } from "@/lib/store/auth";

export interface ResumeProject {
  id: string;
  resume_profile_id?: string;
  name: string;
  role?: string | null;
  description?: string | null;
  impact?: string | null;
  metrics?: string[];
  tech_stack?: string[];
  start_date?: string | null;
  end_date?: string | null;
  sort_order?: number;
}

export interface ResumeProjectUpsert {
  name: string;
  role?: string;
  description?: string;
  impact?: string;
  metrics?: string[];
  tech_stack?: string[];
  start_date?: string;
  end_date?: string;
  sort_order?: number;
}

export interface ResumeProfile {
  id: string;
  user_id: string;
  headline?: string | null;
  summary?: string | null;
  years_experience?: number | null;
  skills?: string[];
  target_keywords?: string[];
  ats_score?: number | null;
  last_scored_at?: string | null;
  projects?: ResumeProject[];
}

export interface ResumeProfileUpsert {
  headline?: string;
  summary?: string;
  years_experience?: number;
  skills?: string[];
  target_keywords?: string[];
}

export interface ResumeScoreResponse {
  ats_score: number;
  keyword_matches?: string[];
  missing_keywords?: string[];
  suggestions?: string[];
  used_fallback?: boolean;
}

/** GET /resume/profile — throws ApiError(404) when not yet created. */
export function getResumeProfile(): Promise<ResumeProfile> {
  return api.get<ResumeProfile>("/resume/profile");
}

/** PUT /resume/profile — create or update the resume profile. */
export function upsertResumeProfile(payload: ResumeProfileUpsert): Promise<ResumeProfile> {
  return api.put<ResumeProfile>("/resume/profile", payload);
}

/** GET /resume/projects. */
export function listResumeProjects(): Promise<ResumeProject[]> {
  return api.get<ResumeProject[]>("/resume/projects");
}

/** POST /resume/projects. */
export function createResumeProject(payload: ResumeProjectUpsert): Promise<ResumeProject> {
  return api.post<ResumeProject>("/resume/projects", payload);
}

/** PUT /resume/projects/{id}. */
export function updateResumeProject(
  id: string,
  payload: ResumeProjectUpsert,
): Promise<ResumeProject> {
  return api.put<ResumeProject>(`/resume/projects/${id}`, payload);
}

/** DELETE /resume/projects/{id}. */
export function deleteResumeProject(id: string): Promise<void> {
  return api.delete<void>(`/resume/projects/${id}`);
}

/** POST /resume/score — ATS + AI review. */
export function scoreResume(): Promise<ResumeScoreResponse> {
  return api.post<ResumeScoreResponse>("/resume/score");
}

/* --------------------------------------------------------------------------
 * Resume file (uploaded PDF/DOCX). Multipart upload and authenticated blob
 * download bypass the JSON `api` client — they use raw fetch with the bearer
 * token read from the auth store (the same source the client's token provider
 * uses).
 * ------------------------------------------------------------------------ */

export interface ResumeFileMeta {
  file_name: string;
  content_type: string;
  size_bytes: number;
  uploaded_at: string;
}

/** Throw a typed ApiError from a non-2xx raw fetch response. */
async function throwApiError(res: Response): Promise<never> {
  let envelope: ApiErrorEnvelope | undefined;
  try {
    envelope = (await res.json()) as ApiErrorEnvelope;
  } catch {
    envelope = undefined;
  }
  const err = envelope?.error;
  throw new ApiError(
    res.status,
    err?.code ?? "INTERNAL",
    err?.message ?? res.statusText ?? "Request failed.",
    err?.details,
    err?.request_id,
  );
}

function authHeaders(): Headers {
  const headers = new Headers();
  const token = useAuthStore.getState().accessToken;
  if (token) headers.set("Authorization", `Bearer ${token}`);
  return headers;
}

/** POST /resume/file — multipart upload of a PDF/DOCX resume file. */
export async function uploadResumeFile(file: File): Promise<ResumeFileMeta> {
  const form = new FormData();
  form.append("file", file);
  const headers = authHeaders();
  headers.set("Accept", "application/json");
  const res = await fetch(`${API_BASE}/resume/file`, {
    method: "POST",
    headers, // do NOT set Content-Type — the browser sets the multipart boundary
    body: form,
    credentials: "include",
  });
  if (!res.ok) await throwApiError(res);
  return (await res.json()) as ResumeFileMeta;
}

/** GET /resume/file/meta — current resume file metadata (404 when none). */
export function getResumeFileMeta(): Promise<ResumeFileMeta> {
  return api.get<ResumeFileMeta>("/resume/file/meta");
}

/** DELETE /resume/file. */
export function deleteResumeFile(): Promise<void> {
  return api.delete<void>("/resume/file");
}

/**
 * GET /resume/file — fetch the bytes (authenticated) and trigger a browser
 * download via an object URL. `fileName` names the saved file.
 */
export async function downloadResumeFile(fileName: string): Promise<void> {
  const res = await fetch(`${API_BASE}/resume/file`, {
    method: "GET",
    headers: authHeaders(),
    credentials: "include",
  });
  if (!res.ok) await throwApiError(res);
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  try {
    const a = document.createElement("a");
    a.href = url;
    a.download = fileName || "resume";
    document.body.appendChild(a);
    a.click();
    a.remove();
  } finally {
    URL.revokeObjectURL(url);
  }
}
