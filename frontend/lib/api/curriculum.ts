/**
 * Curriculum API layer — today's plan, roadmap, plan days, and task lifecycle.
 * Shapes mirror the OpenAPI schemas: PlanDay, PlanTask, Roadmap, RoadmapWeek,
 * CompleteTaskRequest/Response, SkipTaskRequest, RescheduleTaskRequest.
 */

import { api } from "@/lib/api/client";
import type {
  ConfidenceLevel,
  Difficulty,
  PillarType,
  PlanItemType,
  Priority,
  TaskKind,
  TaskStatus,
} from "@/lib/api/types";

export interface PlanTask {
  id: string;
  plan_day_id?: string;
  kind: TaskKind;
  item_type: PlanItemType;
  item_id: string;
  pillar_type: PillarType;
  title: string;
  description?: string | null;
  objectives?: string[];
  estimated_minutes?: number;
  priority?: Priority;
  difficulty?: Difficulty | null;
  status: TaskStatus;
  sort_order?: number;
  confidence?: ConfidenceLevel | null;
  time_spent_minutes?: number | null;
  completion_notes?: string | null;
  revision_item_id?: string | null;
  completed_at?: string | null;
}

export interface PlanDay {
  id: string;
  roadmap_week_id?: string;
  user_id?: string;
  date: string;
  planned_minutes: number;
  completed_minutes?: number;
  is_rest_day?: boolean;
  summary?: string | null;
  tasks?: PlanTask[];
}

export interface RoadmapWeek {
  id: string;
  roadmap_id: string;
  week_number: number;
  start_date: string;
  end_date: string;
  theme?: string | null;
  focus_pillars?: PillarType[];
  planned_hours?: number;
  days?: PlanDay[];
}

export interface Roadmap {
  id: string;
  user_id: string;
  track_id: string;
  profile_id?: string;
  target_company_id?: string | null;
  start_date: string;
  end_date: string;
  total_weeks: number;
  hours_per_week?: number;
  status?: "active" | "completed" | "archived";
  is_active: boolean;
  generated_by?: "engine" | "ai";
  weeks?: RoadmapWeek[];
  created_at?: string;
  updated_at?: string;
}

export interface RevisionItem {
  id: string;
  user_id: string;
  item_type: PlanItemType;
  item_id: string;
  pillar_type: PillarType;
  title?: string;
  interval_days: number;
  stage?: number;
  due_at: string;
}

export interface StreakResponse {
  current_streak: number;
  longest_streak: number;
  days?: Array<{
    date: string;
    tasks_completed: number;
    minutes_studied: number;
    goal_met: boolean;
  }>;
}

export interface CompleteTaskRequest {
  confidence: ConfidenceLevel;
  time_spent_minutes: number;
  notes?: string;
}

export interface CompleteTaskResponse {
  task: PlanTask;
  revision_item?: RevisionItem | null;
  streak?: StreakResponse;
}

export interface SkipTaskRequest {
  reason?: string;
}

export interface RescheduleTaskRequest {
  to_date: string;
}

/** GET /today — the auto-generated plan day for today. 404 when no roadmap. */
export function getToday(): Promise<PlanDay> {
  return api.get<PlanDay>("/today");
}

/** GET /plan-days/{date} — the plan day for a specific date (YYYY-MM-DD). */
export function getPlanDay(date: string): Promise<PlanDay> {
  return api.get<PlanDay>(`/plan-days/${date}`);
}

/** GET /roadmaps/active — active roadmap with week summaries. 404 when none. */
export function getActiveRoadmap(): Promise<Roadmap> {
  return api.get<Roadmap>("/roadmaps/active");
}

/**
 * POST /roadmaps/generate — build (or regenerate) the active roadmap from the
 * user's saved profile. Called after intake so a plan actually exists; pass
 * `regenerate: true` to rebuild an existing one.
 */
export function generateRoadmap(body: { regenerate?: boolean } = {}): Promise<Roadmap> {
  return api.post<Roadmap>("/roadmaps/generate", body);
}

/** GET /roadmaps/{roadmapId}/weeks/{weekNumber} — a week with its days/tasks. */
export function getRoadmapWeek(roadmapId: string, weekNumber: number): Promise<RoadmapWeek> {
  return api.get<RoadmapWeek>(`/roadmaps/${roadmapId}/weeks/${weekNumber}`);
}

/** POST /tasks/{taskId}/complete — confidence + time + optional notes. */
export function completeTask(
  taskId: string,
  body: CompleteTaskRequest,
): Promise<CompleteTaskResponse> {
  return api.post<CompleteTaskResponse>(`/tasks/${taskId}/complete`, body);
}

/** POST /tasks/{taskId}/start — mark a task in progress. */
export function startTask(taskId: string): Promise<PlanTask> {
  return api.post<PlanTask>(`/tasks/${taskId}/start`);
}

/** POST /tasks/{taskId}/reopen — revert a completed/skipped task to pending. */
export function reopenTask(taskId: string): Promise<PlanTask> {
  return api.post<PlanTask>(`/tasks/${taskId}/reopen`);
}

/** POST /tasks/{taskId}/skip — skip with optional reason. */
export function skipTask(taskId: string, body: SkipTaskRequest = {}): Promise<PlanTask> {
  return api.post<PlanTask>(`/tasks/${taskId}/skip`, body);
}

/** POST /tasks/{taskId}/reschedule — move a task to another date. */
export function rescheduleTask(
  taskId: string,
  body: RescheduleTaskRequest,
): Promise<PlanTask> {
  return api.post<PlanTask>(`/tasks/${taskId}/reschedule`, body);
}
