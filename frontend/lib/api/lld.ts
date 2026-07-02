/**
 * LLD-problem admin authoring API — create / update / delete (admin-only; the
 * catalog GETs live in lib/api/content.ts). The request body mirrors the
 * LLDProblemWrite schema in docs/openapi.yaml exactly.
 */

import { api } from "@/lib/api/client";
import type { LLDProblemDetail } from "@/lib/api/content";
import type { Difficulty } from "@/lib/api/types";

/** Request body for POST/PUT /lld-problems (schema: LLDProblemWrite). */
export interface LLDProblemWrite {
  track_id?: string | null;
  pillar_id?: string | null;
  slug: string;
  title: string;
  difficulty: Difficulty;
  order_index?: number | null;
  requirements_md?: string | null;
  entities_md?: string | null;
  class_diagram_md?: string | null;
  design_patterns?: string[];
  solid_notes_md?: string | null;
  api_or_interface_md?: string | null;
  tradeoffs_md?: string | null;
  follow_up_questions?: string[];
}

/** POST /lld-problems — create an LLD problem (admin). */
export function createLLDProblem(body: LLDProblemWrite): Promise<LLDProblemDetail> {
  return api.post<LLDProblemDetail>("/lld-problems", body);
}

/** PUT /lld-problems/{id} — replace an LLD problem (admin). */
export function updateLLDProblem(id: string, body: LLDProblemWrite): Promise<LLDProblemDetail> {
  return api.put<LLDProblemDetail>(`/lld-problems/${id}`, body);
}

/** DELETE /lld-problems/{id} — delete an LLD problem (admin; 204). */
export function deleteLLDProblem(id: string): Promise<void> {
  return api.delete<void>(`/lld-problems/${id}`);
}
