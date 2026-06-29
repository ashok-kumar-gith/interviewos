/**
 * Dashboard analytics API layer (GET /dashboard → DashboardResponse).
 * Shapes mirror the OpenAPI DashboardResponse schema.
 */

import { api } from "@/lib/api/client";
import type { PillarType } from "@/lib/api/types";

export interface PillarReadiness {
  pillar: PillarType;
  readiness: number;
  coverage: number;
  avg_confidence: number;
  revision_health: number;
}

export interface DashboardTodaySummary {
  date: string;
  total_tasks: number;
  completed_tasks: number;
  estimated_hours: number;
  remaining_hours: number;
}

export interface DashboardResponse {
  overall_readiness: number;
  estimated_readiness_date?: string | null;
  pillar_readiness: PillarReadiness[];
  study_streak: {
    current: number;
    longest: number;
  };
  today: DashboardTodaySummary;
  revision_due_count: number;
  generated_at: string;
}

/** GET /dashboard — aggregate readiness, streak, today summary. */
export function getDashboard(): Promise<DashboardResponse> {
  return api.get<DashboardResponse>("/dashboard");
}
