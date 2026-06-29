/**
 * Resume API layer — profile, projects CRUD, and ATS scoring.
 * Shapes mirror the OpenAPI schemas: ResumeProfile, ResumeProfileUpsert,
 * ResumeProject, ResumeProjectUpsert, ResumeScoreResponse.
 */

import { api } from "@/lib/api/client";

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
