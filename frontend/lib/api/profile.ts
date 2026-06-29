/**
 * Profile / intake API layer — typed wrappers over the InterviewOS profile and
 * company endpoints. Shapes mirror the OpenAPI schemas (docs/openapi.yaml):
 * UserProfile, UserProfileUpsert, Company, Track.
 *
 * `apiFetch` unwraps the success envelope, so list endpoints that return a
 * `PaginatedEnvelope` resolve to the inner `data` array directly.
 */

import { api } from "@/lib/api/client";
import type { PillarType } from "@/lib/intake/pillars";

/** ConfidenceLevel — integer 1–5 (self-assessed pillar strength). */
export type ConfidenceLevel = 1 | 2 | 3 | 4 | 5;

/** A self-assessment value per pillar (keys are PillarType enum values). */
export type PillarStrengths = Partial<Record<PillarType, ConfidenceLevel>>;

/** GET /profile — the user's saved intake profile. */
export interface UserProfile {
  id: string;
  user_id: string;
  track_id: string;
  years_experience?: number;
  target_company_id?: string | null;
  target_role: string;
  target_level?: string | null;
  hours_per_week: number;
  start_date: string; // YYYY-MM-DD
  target_weeks: number;
  pillar_strengths?: PillarStrengths;
  timezone?: string;
  onboarding_completed_at?: string | null;
  created_at: string;
  updated_at: string;
}

/** PUT /profile request body (UserProfileUpsert). */
export interface UserProfileUpsert {
  track_id: string;
  years_experience?: number;
  target_company_id?: string | null;
  target_role: string;
  target_level?: string | null;
  hours_per_week: number;
  start_date: string; // YYYY-MM-DD
  target_weeks?: number;
  pillar_strengths?: PillarStrengths;
  timezone?: string;
  intake_answers?: Record<string, unknown>;
}

/** GET /companies item (Company). */
export interface Company {
  id: string;
  slug: string;
  name: string;
  logo_url?: string | null;
  description?: string | null;
  is_fully_weighted?: boolean;
}

/** GET /tracks item (Track) — needed for the required `track_id` on upsert. */
export interface Track {
  id: string;
  slug: string;
  name: string;
  description?: string | null;
  seniority?: string | null;
  is_active?: boolean;
  sort_order?: number;
}

/** Fetch the current intake profile. Throws ApiError(404) when not yet created. */
export function getProfile(): Promise<UserProfile> {
  return api.get<UserProfile>("/profile");
}

/** Create or update the intake profile (idempotent upsert). */
export function upsertProfile(payload: UserProfileUpsert): Promise<UserProfile> {
  return api.put<UserProfile>("/profile", payload);
}

/** List companies for the target-company select. */
export function listCompanies(query?: string): Promise<Company[]> {
  return api.get<Company[]>("/companies", {
    query: { page_size: 100, q: query || undefined },
  });
}

/** List tracks — the upsert requires a `track_id`. */
export function listTracks(): Promise<Track[]> {
  return api.get<Track[]>("/tracks", { query: { page_size: 100 } });
}
