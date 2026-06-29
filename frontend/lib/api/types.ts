/**
 * Shared API enum/value types mirroring the OpenAPI schemas (docs/openapi.yaml).
 * Plain types only — safe to import from both Server and Client components.
 */

import type { PillarType } from "@/lib/intake/pillars";

export type { PillarType };

export type Difficulty = "easy" | "medium" | "hard";
export type Priority = "low" | "medium" | "high" | "critical";
export type ProgressStatus =
  | "not_started"
  | "in_progress"
  | "completed"
  | "needs_review";
export type ResourceType =
  | "book"
  | "video"
  | "article"
  | "course"
  | "github"
  | "practice"
  | "documentation"
  | "blog"
  | "cheatsheet";
export type TaskKind = "study" | "solve" | "read" | "watch" | "revise" | "mock";
export type TaskStatus =
  | "pending"
  | "in_progress"
  | "completed"
  | "skipped"
  | "rescheduled";
export type PlanItemType =
  | "topic"
  | "subtopic"
  | "problem"
  | "resource"
  | "design_problem"
  | "lld_problem"
  | "behavioral_story"
  | "revision_item";
export type ConfidenceLevel = 1 | 2 | 3 | 4 | 5;
export type ProblemSourceName =
  | "blind75"
  | "neetcode150"
  | "grind75"
  | "tech_interview_handbook"
  | "leetcode_top"
  | "striver_sde"
  | "custom";
export type ProblemPlatform =
  | "leetcode"
  | "hackerrank"
  | "codeforces"
  | "interviewbit"
  | "gfg"
  | "custom";

/** Pagination meta from the PaginatedEnvelope. */
export interface PaginationMeta {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

/**
 * A paginated list result. `apiFetch` unwraps the success envelope's `data`,
 * so for list endpoints we ask the client for the full envelope (data + meta)
 * via a dedicated helper rather than the default unwrap. See `listPaginated`.
 */
export interface Paginated<T> {
  data: T[];
  meta: PaginationMeta;
}
