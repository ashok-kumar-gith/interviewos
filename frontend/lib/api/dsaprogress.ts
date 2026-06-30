/**
 * DSA problem progress + stored solution API. The problem catalog is read via
 * lib/api/content.ts; this module covers marking a problem solved, when it was
 * solved, and saving the solution the user wrote (code + language + note).
 */

import { api } from "@/lib/api/client";

export type ProgressStatus = "not_started" | "in_progress" | "completed" | "needs_review";

export interface ProblemProgress {
  problem_id: string;
  status: ProgressStatus;
  solved: boolean;
  confidence?: number | null;
  attempts: number;
  time_spent_minutes: number;
  solved_at?: string | null;
  solution_code?: string | null;
  solution_language?: string | null;
  solution_notes?: string | null;
  solution_updated_at?: string | null;
  updated_at?: string;
}

export interface SaveProblemProgress {
  solved: boolean;
  confidence?: number | null;
  time_spent_minutes?: number;
  solution_code?: string | null;
  solution_language?: string | null;
  solution_notes?: string | null;
}

/** GET /problems/{id}/progress — progress + stored solution (not_started when none). */
export function getProblemProgress(id: string): Promise<ProblemProgress> {
  return api.get<ProblemProgress>(`/problems/${id}/progress`);
}

/** PUT /problems/{id}/progress — record solve state + solution. */
export function saveProblemProgress(id: string, body: SaveProblemProgress): Promise<ProblemProgress> {
  return api.put<ProblemProgress>(`/problems/${id}/progress`, body);
}

/** DELETE /problems/{id}/progress — clear progress + solution. */
export function deleteProblemProgress(id: string): Promise<void> {
  return api.delete<void>(`/problems/${id}/progress`);
}

/** GET /problems/solved — the user's solved/attempted log. */
export function listSolvedProblems(): Promise<ProblemProgress[]> {
  return api.get<{ data: ProblemProgress[] }>("/problems/solved").then((r) => r.data ?? []);
}
