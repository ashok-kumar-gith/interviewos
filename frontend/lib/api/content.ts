/**
 * Content catalog API layer — DSA problems, patterns, topics, design problems,
 * LLD problems, companies. Shapes mirror the OpenAPI schemas. List endpoints
 * return the pagination envelope ({ data, meta }) via `api.getList`.
 */

import { api } from "@/lib/api/client";
import type {
  ConfidenceLevel,
  Difficulty,
  PaginationMeta,
  PillarType,
  Priority,
  ProblemPlatform,
  ProblemSourceName,
  ProgressStatus,
  ResourceType,
} from "@/lib/api/types";

/* ---- DSA ---- */

export interface Problem {
  id: string;
  track_id: string;
  topic_id?: string | null;
  slug: string;
  title: string;
  difficulty: Difficulty;
  platform?: ProblemPlatform;
  external_id?: string | null;
  url?: string | null;
  estimated_minutes?: number;
  frequency_score?: number;
  is_premium?: boolean;
}

export interface Pattern {
  id: string;
  track_id?: string;
  slug: string;
  name: string;
  description?: string | null;
  when_to_use?: string | null;
}

/* ---- Content ---- */

export interface Topic {
  id: string;
  pillar_id: string;
  track_id: string;
  slug: string;
  name: string;
  summary?: string | null;
  difficulty: Difficulty;
  priority: Priority;
  estimated_hours?: number;
  sort_order?: number;
}

/* ---- Design / LLD ---- */

export interface DesignProblem {
  id: string;
  track_id?: string;
  slug: string;
  title: string;
  difficulty: Difficulty;
  order_index: number;
}

export interface LLDProblem {
  id: string;
  track_id?: string;
  slug: string;
  title: string;
  difficulty: Difficulty;
  order_index: number;
}

/* ---- Company ---- */

export interface Company {
  id: string;
  slug: string;
  name: string;
  logo_url?: string | null;
  description?: string | null;
  is_fully_weighted?: boolean;
}

/* ---- Resources ---- */

export interface Resource {
  id: string;
  type: ResourceType;
  title: string;
  author?: string | null;
  url?: string | null;
  provider?: string | null;
  description?: string | null;
  estimated_minutes?: number | null;
  difficulty?: Difficulty | null;
  priority?: Priority;
  is_free?: boolean;
}

/* ---- Progress (embedded in detail responses) ---- */

export interface UserTopicProgress {
  id: string;
  user_id: string;
  topic_id: string;
  status: ProgressStatus;
  confidence?: ConfidenceLevel | null;
}

export interface UserProblemProgress {
  id: string;
  user_id: string;
  problem_id: string;
  status: ProgressStatus;
  solved: boolean;
  confidence?: ConfidenceLevel | null;
}

/* ---- Detail shapes (full content for the [id] pages) ---- */

export interface Subtopic {
  id: string;
  topic_id: string;
  slug: string;
  name: string;
  content_md?: string | null;
  estimated_hours?: number;
  sort_order?: number;
}

export interface TopicDetail extends Topic {
  concept_md?: string | null;
  common_mistakes?: string | null;
  expected_questions?: string[];
  prerequisites?: string[];
  subtopics?: Subtopic[];
  resources?: Resource[];
  progress?: UserTopicProgress | null;
}

export interface ProblemSource {
  source: ProblemSourceName;
  source_rank?: number | null;
  source_url?: string | null;
}

export interface ProblemCompanyFrequency {
  company_id: string;
  company_name: string;
  frequency: number;
  last_seen_period?: string | null;
}

export interface ProblemDetail extends Problem {
  prompt_summary?: string | null;
  approach_md?: string | null;
  common_mistakes?: string | null;
  patterns?: Pattern[];
  sources?: ProblemSource[];
  company_frequency?: ProblemCompanyFrequency[];
  progress?: UserProblemProgress | null;
}

export interface DesignProblemDetail extends DesignProblem {
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

export interface LLDProblemDetail extends LLDProblem {
  requirements_md?: string | null;
  entities_md?: string | null;
  class_diagram_md?: string | null;
  design_patterns?: string[];
  solid_notes_md?: string | null;
  api_or_interface_md?: string | null;
  tradeoffs_md?: string | null;
  follow_up_questions?: string[];
}

export interface ListResult<T> {
  data: T[];
  meta: Partial<PaginationMeta>;
}

export interface ProblemFilters {
  page?: number;
  page_size?: number;
  difficulty?: Difficulty;
  pattern_id?: string;
  topic_id?: string;
  company_id?: string;
  source?: ProblemSourceName;
  solved?: boolean;
  q?: string;
  sort?: string;
}

/** GET /problems — paginated DSA problems with filters. */
export function listProblems(filters: ProblemFilters = {}): Promise<ListResult<Problem>> {
  return api.getList<Problem>("/problems", { query: { ...filters } });
}

/** GET /patterns — DSA patterns (used to populate the pattern filter). */
export function listPatterns(query?: string): Promise<ListResult<Pattern>> {
  return api.getList<Pattern>("/patterns", { query: { page_size: 100, q: query || undefined } });
}

/** GET /topics — paginated topics. */
export function listTopics(filters: {
  page?: number;
  page_size?: number;
  difficulty?: Difficulty;
  priority?: Priority;
  q?: string;
} = {}): Promise<ListResult<Topic>> {
  return api.getList<Topic>("/topics", { query: { ...filters } });
}

/** GET /design-problems — HLD design problems (ordered catalog). */
export function listDesignProblems(filters: {
  page?: number;
  page_size?: number;
  difficulty?: Difficulty;
  q?: string;
} = {}): Promise<ListResult<DesignProblem>> {
  return api.getList<DesignProblem>("/design-problems", {
    query: { sort: "order_index", ...filters },
  });
}

/** GET /lld-problems — LLD problems. */
export function listLLDProblems(filters: {
  page?: number;
  page_size?: number;
  difficulty?: Difficulty;
  q?: string;
} = {}): Promise<ListResult<LLDProblem>> {
  return api.getList<LLDProblem>("/lld-problems", {
    query: { sort: "order_index", ...filters },
  });
}

/** GET /companies — used for the problems company filter. */
export function listCompanies(query?: string): Promise<ListResult<Company>> {
  return api.getList<Company>("/companies", {
    query: { page_size: 100, q: query || undefined },
  });
}

/* ---- Detail fetchers ---- */

/** GET /problems/{id} — a problem with patterns, sources, company frequency. */
export function getProblem(id: string): Promise<ProblemDetail> {
  return api.get<ProblemDetail>(`/problems/${id}`);
}

/** GET /design-problems/{id} — an HLD design problem with all sections. */
export function getDesignProblem(id: string): Promise<DesignProblemDetail> {
  return api.get<DesignProblemDetail>(`/design-problems/${id}`);
}

/** GET /lld-problems/{id} — an LLD problem with all sections. */
export function getLLDProblem(id: string): Promise<LLDProblemDetail> {
  return api.get<LLDProblemDetail>(`/lld-problems/${id}`);
}

/** GET /topics/{id} — a topic with subtopics, resources, and progress. */
export function getTopic(id: string): Promise<TopicDetail> {
  return api.get<TopicDetail>(`/topics/${id}`);
}

/* ---- Backend Engineering (pillar-scoped topics) ---- */

/** GET /backend-engineering/topics — paginated backend engineering topics. */
export function listBackendEngineeringTopics(filters: {
  page?: number;
  page_size?: number;
  difficulty?: Difficulty;
  priority?: Priority;
  q?: string;
} = {}): Promise<ListResult<Topic>> {
  return api.getList<Topic>("/backend-engineering/topics", { query: { ...filters } });
}

/** GET /backend-engineering/topics/{id} — a backend engineering topic detail. */
export function getBackendEngineeringTopic(id: string): Promise<TopicDetail> {
  return api.get<TopicDetail>(`/backend-engineering/topics/${id}`);
}

export interface ResourceFilters {
  page?: number;
  page_size?: number;
  type?: ResourceType;
  topic_id?: string;
  difficulty?: Difficulty;
  q?: string;
  sort?: string;
}

/** GET /resources — paginated resource library with type/difficulty filters. */
export function listResources(filters: ResourceFilters = {}): Promise<ListResult<Resource>> {
  return api.getList<Resource>("/resources", { query: { ...filters } });
}

/* ==========================================================================
 * Admin authoring — create / update / delete (admin-only; GET stays public).
 * Request bodies mirror the *Write schemas in docs/openapi.yaml exactly.
 * ======================================================================== */

/** A single company-frequency entry on a ProblemWrite (ProblemCompanyFrequencyWrite). */
export interface ProblemCompanyFrequencyWrite {
  company_id?: string | null;
  company_slug?: string;
  frequency: number;
  last_seen_period?: string | null;
}

/** Request body for POST/PUT /problems (schema: ProblemWrite). */
export interface ProblemWrite {
  track_id?: string | null;
  topic_id?: string | null;
  slug: string;
  title: string;
  difficulty: Difficulty;
  platform?: ProblemPlatform;
  external_id?: string | null;
  url?: string | null;
  prompt_summary?: string | null;
  approach_md?: string | null;
  common_mistakes?: string | null;
  estimated_minutes?: number | null;
  frequency_score?: number | null;
  is_premium?: boolean | null;
  pattern_slugs?: string[];
  sources?: ProblemSourceName[];
  company_frequency?: ProblemCompanyFrequencyWrite[];
}

/** POST /problems — create a DSA problem (admin). Returns the created problem. */
export function createProblem(body: ProblemWrite): Promise<ProblemDetail> {
  return api.post<ProblemDetail>("/problems", body);
}

/** PUT /problems/{id} — replace a DSA problem (admin). */
export function updateProblem(id: string, body: ProblemWrite): Promise<ProblemDetail> {
  return api.put<ProblemDetail>(`/problems/${id}`, body);
}

/** DELETE /problems/{id} — delete a DSA problem (admin; 204). */
export function deleteProblem(id: string): Promise<void> {
  return api.delete<void>(`/problems/${id}`);
}

/** Request body for POST/PUT /topics (schema: TopicWrite). */
export interface TopicWrite {
  pillar_id?: string | null;
  pillar_type?: PillarType;
  track_id?: string | null;
  slug: string;
  name: string;
  summary?: string | null;
  concept_md?: string | null;
  difficulty?: Difficulty;
  priority?: Priority;
  estimated_hours?: number | null;
  common_mistakes?: string | null;
  expected_questions?: string[];
  prerequisites?: string[];
  sort_order?: number | null;
}

/** POST /topics — create a topic under a pillar (admin). */
export function createTopic(body: TopicWrite): Promise<TopicDetail> {
  return api.post<TopicDetail>("/topics", body);
}

/** PUT /topics/{id} — replace a topic (admin). */
export function updateTopic(id: string, body: TopicWrite): Promise<TopicDetail> {
  return api.put<TopicDetail>(`/topics/${id}`, body);
}

/** DELETE /topics/{id} — delete a topic (admin; 204). */
export function deleteTopic(id: string): Promise<void> {
  return api.delete<void>(`/topics/${id}`);
}
