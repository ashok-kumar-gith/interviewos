/**
 * Design-problem (HLD) per-user progress API. The catalog itself is read via
 * lib/api/content.ts (listDesignProblems/getDesignProblem); this module covers
 * marking a design problem done + confidence, mirroring DSA problem tracking.
 */

import { api } from "@/lib/api/client";
import type { DesignProblemDetail } from "@/lib/api/content";
import type { Difficulty } from "@/lib/api/types";

export type ProgressStatus = "not_started" | "in_progress" | "completed" | "needs_review";

/* ==========================================================================
 * Admin authoring — create / update / delete HLD design problems (admin-only).
 * Request body mirrors the DesignProblemWrite schema in docs/openapi.yaml.
 * ======================================================================== */

/** Request body for POST/PUT /design-problems (schema: DesignProblemWrite). */
export interface DesignProblemWrite {
  track_id?: string | null;
  pillar_id?: string | null;
  slug: string;
  title: string;
  difficulty: Difficulty;
  order_index?: number | null;
  requirements_md?: string | null;
  capacity_estimation_md?: string | null;
  api_design_md?: string | null;
  data_model_md?: string | null;
  high_level_design_md?: string | null;
  caching_md?: string | null;
  queueing_md?: string | null;
  scaling_md?: string | null;
  tradeoffs_md?: string | null;
  failure_handling_md?: string | null;
  alternatives_md?: string | null;
  interview_tips_md?: string | null;
  follow_up_questions?: string[];
}

/** POST /design-problems — create an HLD design problem (admin). */
export function createDesignProblem(body: DesignProblemWrite): Promise<DesignProblemDetail> {
  return api.post<DesignProblemDetail>("/design-problems", body);
}

/** PUT /design-problems/{id} — replace an HLD design problem (admin). */
export function updateDesignProblem(
  id: string,
  body: DesignProblemWrite,
): Promise<DesignProblemDetail> {
  return api.put<DesignProblemDetail>(`/design-problems/${id}`, body);
}

/** DELETE /design-problems/{id} — delete an HLD design problem (admin; 204). */
export function deleteDesignProblem(id: string): Promise<void> {
  return api.delete<void>(`/design-problems/${id}`);
}

export interface DesignProblemProgress {
  design_problem_id: string;
  status: ProgressStatus;
  confidence?: number | null;
  attempts: number;
  time_spent_minutes: number;
  notes?: string | null;
  first_completed_at?: string | null;
  updated_at?: string;
}

export interface SaveDesignProblemProgress {
  status: ProgressStatus;
  confidence?: number | null;
  time_spent_minutes?: number;
  notes?: string | null;
}

/** GET /design-problems/{id}/progress — the caller's progress (not_started when none). */
export function getDesignProblemProgress(id: string): Promise<DesignProblemProgress> {
  return api.get<DesignProblemProgress>(`/design-problems/${id}/progress`);
}

/** PUT /design-problems/{id}/progress — record status + confidence. */
export function saveDesignProblemProgress(
  id: string,
  body: SaveDesignProblemProgress,
): Promise<DesignProblemProgress> {
  return api.put<DesignProblemProgress>(`/design-problems/${id}/progress`, body);
}

/** DELETE /design-problems/{id}/progress — clear the caller's progress (204). */
export function deleteDesignProblemProgress(id: string): Promise<void> {
  return api.delete<void>(`/design-problems/${id}/progress`);
}
