/**
 * Revision API layer — list due items and record recall outcomes.
 * Shapes mirror the backend `RevisionItem` (internal/revision/handler.go).
 */

import { api } from "@/lib/api/client";
import type { PillarType } from "@/lib/api/types";

export type RecallOutcome = "correct" | "incorrect";

export interface RevisionItem {
  id: string;
  user_id: string;
  item_type: string;
  item_id: string;
  pillar_type: PillarType;
  /** May be empty/omitted by the backend — fall back to a humanized item_type. */
  title?: string;
  interval_days: number;
  stage: number;
  ease: number;
  due_at: string;
  last_reviewed_at?: string | null;
  last_recall?: RecallOutcome | null;
  review_count: number;
  lapse_count: number;
  is_active: boolean;
}

/** GET /revision/due — items currently due for revision. */
export function getDueRevisions(): Promise<RevisionItem[]> {
  return api
    .getList<RevisionItem>("/revision/due", {
      query: { page_size: 50 },
    })
    .then((r) => r.data);
}

/** POST /revision/{id}/recall — record a recall outcome for a due item. */
export function recordRecall(
  id: string,
  recall: RecallOutcome,
  timeSpentMinutes = 0,
): Promise<RevisionItem> {
  return api.post<RevisionItem>(`/revision/${id}/recall`, {
    recall,
    time_spent_minutes: timeSpentMinutes,
  });
}
