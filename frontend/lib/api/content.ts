/**
 * Content catalog API layer — DSA problems, patterns, topics, design problems,
 * LLD problems, companies. Shapes mirror the OpenAPI schemas. List endpoints
 * return the pagination envelope ({ data, meta }) via `api.getList`.
 */

import { api } from "@/lib/api/client";
import type {
  Difficulty,
  PaginationMeta,
  Priority,
  ProblemPlatform,
  ProblemSourceName,
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
