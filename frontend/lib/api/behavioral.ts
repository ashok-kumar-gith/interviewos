/**
 * Behavioral (STAR stories) API layer — CRUD + AI improve.
 * Shapes mirror the OpenAPI schemas: BehavioralStory, BehavioralStoryUpsert,
 * StoryImproveResponse, StoryTheme.
 */

import { api } from "@/lib/api/client";

export type StoryTheme =
  | "leadership"
  | "ownership"
  | "conflict"
  | "failure"
  | "mentorship"
  | "stakeholder_management"
  | "project_rescue"
  | "production_incident"
  | "ambiguity"
  | "impact";

export const STORY_THEMES: { value: StoryTheme; label: string }[] = [
  { value: "leadership", label: "Leadership" },
  { value: "ownership", label: "Ownership" },
  { value: "conflict", label: "Conflict" },
  { value: "failure", label: "Failure" },
  { value: "mentorship", label: "Mentorship" },
  { value: "stakeholder_management", label: "Stakeholder management" },
  { value: "project_rescue", label: "Project rescue" },
  { value: "production_incident", label: "Production incident" },
  { value: "ambiguity", label: "Ambiguity" },
  { value: "impact", label: "Impact" },
];

export function themeLabel(theme: StoryTheme): string {
  return STORY_THEMES.find((t) => t.value === theme)?.label ?? theme;
}

export interface BehavioralStory {
  id: string;
  user_id: string;
  title: string;
  theme: StoryTheme;
  situation?: string | null;
  task?: string | null;
  action?: string | null;
  result?: string | null;
  metrics?: string | null;
  tags?: string[];
  ai_improved?: boolean;
  strength_score?: number | null;
  created_at?: string;
  updated_at?: string;
}

export interface BehavioralStoryUpsert {
  title: string;
  theme: StoryTheme;
  situation?: string;
  task?: string;
  action?: string;
  result?: string;
  metrics?: string;
  tags?: string[];
}

export interface StoryImproveResponse {
  story_id: string;
  improved?: {
    situation?: string;
    task?: string;
    action?: string;
    result?: string;
    metrics?: string;
  };
  suggestions: string[];
  strength_score?: number;
  used_fallback?: boolean;
}

/** GET /behavioral-stories — optionally filtered by theme. */
export function listStories(theme?: StoryTheme): Promise<BehavioralStory[]> {
  return api
    .getList<BehavioralStory>("/behavioral-stories", {
      query: { page_size: 100, theme },
    })
    .then((r) => r.data);
}

/** POST /behavioral-stories — create a story. */
export function createStory(payload: BehavioralStoryUpsert): Promise<BehavioralStory> {
  return api.post<BehavioralStory>("/behavioral-stories", payload);
}

/** PUT /behavioral-stories/{id} — update a story. */
export function updateStory(id: string, payload: BehavioralStoryUpsert): Promise<BehavioralStory> {
  return api.put<BehavioralStory>(`/behavioral-stories/${id}`, payload);
}

/** DELETE /behavioral-stories/{id}. */
export function deleteStory(id: string): Promise<void> {
  return api.delete<void>(`/behavioral-stories/${id}`);
}

/** POST /behavioral-stories/{id}/improve — AI-improve a story. */
export function improveStory(id: string): Promise<StoryImproveResponse> {
  return api.post<StoryImproveResponse>(`/behavioral-stories/${id}/improve`);
}
