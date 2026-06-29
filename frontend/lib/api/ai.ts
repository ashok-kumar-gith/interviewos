/**
 * AI assistants API layer — planner, coach, resume-review, story-improve,
 * weakness-detect, daily-plan, sd-review. Shapes mirror the OpenAPI schemas:
 * AIResponse, WeaknessDetectResponse, AI*Request. Every response carries
 * `used_fallback` to distinguish a real model answer from a heuristic one.
 */

import { api } from "@/lib/api/client";
import type { PillarType } from "@/lib/api/types";
import type { TopicAnalyticsEntry } from "@/lib/api/analytics";

export type AIFeature =
  | "planner"
  | "coach"
  | "resume_review"
  | "story_improve"
  | "weakness_detect"
  | "daily_plan"
  | "sd_review";

export interface AIResponse {
  feature: AIFeature;
  content: string;
  structured?: Record<string, unknown>;
  model?: string | null;
  used_fallback?: boolean;
  invocation_id?: string;
}

export interface WeaknessDetectResponse {
  weak_topics?: TopicAnalyticsEntry[];
  recommended_tasks?: string[];
  used_fallback?: boolean;
}

export interface AIPlannerRequest {
  roadmap_id?: string;
  focus_pillars?: PillarType[];
  notes?: string;
}

export interface AICoachRequest {
  message: string;
  context_topic_id?: string;
}

export interface AIDailyPlanRequest {
  date?: string;
}

export interface AISdReviewRequest {
  design_problem_id: string;
  answer_md: string;
}

/** POST /ai/planner — AI study planner (refines roadmap suggestions). */
export function aiPlanner(body: AIPlannerRequest = {}): Promise<AIResponse> {
  return api.post<AIResponse>("/ai/planner", body);
}

/** POST /ai/coach — AI interview coach (Q&A on prep). */
export function aiCoach(body: AICoachRequest): Promise<AIResponse> {
  return api.post<AIResponse>("/ai/coach", body);
}

/** POST /ai/coach — ask the AI coach an interview-prep question. */
export function askCoach(message: string, contextTopicId?: string): Promise<AIResponse> {
  return api.post<AIResponse>("/ai/coach", { message, context_topic_id: contextTopicId });
}

/** POST /ai/planner — request AI study-plan suggestions. */
export function requestPlannerSuggestions(input: {
  roadmap_id?: string;
  focus_pillars?: string[];
  notes?: string;
}): Promise<AIResponse> {
  return api.post<AIResponse>("/ai/planner", input);
}

/** POST /ai/weakness-detect — AI weakness detector from progress + mocks. */
export function aiWeaknessDetect(): Promise<WeaknessDetectResponse> {
  return api.post<WeaknessDetectResponse>("/ai/weakness-detect");
}

/** POST /ai/daily-plan — AI daily-plan refinement for a date. */
export function aiDailyPlan(body: AIDailyPlanRequest = {}): Promise<AIResponse> {
  return api.post<AIResponse>("/ai/daily-plan", body);
}

/** POST /ai/sd-review — AI system-design review of a user's design answer. */
export function aiSdReview(body: AISdReviewRequest): Promise<AIResponse> {
  return api.post<AIResponse>("/ai/sd-review", body);
}
