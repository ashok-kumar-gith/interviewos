/**
 * Design-problem (HLD) per-user progress API. The catalog itself is read via
 * lib/api/content.ts (listDesignProblems/getDesignProblem); this module covers
 * marking a design problem done + confidence, mirroring DSA problem tracking.
 */

import { api } from "@/lib/api/client";

export type ProgressStatus = "not_started" | "in_progress" | "completed" | "needs_review";

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
