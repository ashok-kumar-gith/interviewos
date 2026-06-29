/**
 * Analytics API layer — readiness, snapshots, streak, topics, time-spent.
 * Shapes mirror the OpenAPI schemas: ReadinessSnapshot, StreakResponse,
 * TopicAnalyticsResponse, TimeSpentResponse.
 */

import { api } from "@/lib/api/client";
import type { ConfidenceLevel, PillarType } from "@/lib/api/types";

export interface ReadinessSnapshot {
  id?: string;
  user_id?: string;
  roadmap_id?: string | null;
  snapshot_date: string;
  overall_readiness: number;
  pillar_readiness?: Record<string, number>;
  completion_pct?: number;
  avg_confidence?: number | null;
  revision_health?: number | null;
  estimated_ready_date?: string | null;
  weak_topics?: string[];
  strong_topics?: string[];
}

export interface StreakDay {
  date: string;
  tasks_completed: number;
  minutes_studied: number;
  goal_met: boolean;
}

export interface StreakResponse {
  current_streak: number;
  longest_streak: number;
  days?: StreakDay[];
}

export interface TopicAnalyticsEntry {
  topic_id: string;
  topic_name: string;
  pillar_type: PillarType;
  confidence?: ConfidenceLevel | null;
  completion_pct?: number;
  revision_accuracy?: number | null;
}

export interface TopicAnalyticsResponse {
  weak?: TopicAnalyticsEntry[];
  strong?: TopicAnalyticsEntry[];
}

export interface TimeSpentBucket {
  key: string;
  minutes: number;
}

export interface TimeSpentResponse {
  total_minutes?: number;
  group_by?: "day" | "pillar";
  buckets?: TimeSpentBucket[];
}

/** GET /analytics/readiness. */
export function getReadiness(): Promise<ReadinessSnapshot> {
  return api.get<ReadinessSnapshot>("/analytics/readiness");
}

/** GET /analytics/snapshots — readiness over time. */
export function getSnapshots(params?: { from?: string; to?: string }): Promise<ReadinessSnapshot[]> {
  return api
    .getList<ReadinessSnapshot>("/analytics/snapshots", {
      query: { page_size: 100, from: params?.from, to: params?.to },
    })
    .then((r) => r.data);
}

/** GET /analytics/streak. */
export function getStreak(params?: { from?: string; to?: string }): Promise<StreakResponse> {
  return api.get<StreakResponse>("/analytics/streak", {
    query: { from: params?.from, to: params?.to },
  });
}

/** GET /analytics/topics — weak/strong topics. */
export function getTopicAnalytics(
  bucket: "weak" | "strong" | "all" = "all",
): Promise<TopicAnalyticsResponse> {
  return api.get<TopicAnalyticsResponse>("/analytics/topics", { query: { bucket } });
}

/** GET /analytics/time-spent. */
export function getTimeSpent(params?: {
  from?: string;
  to?: string;
  group_by?: "day" | "pillar";
}): Promise<TimeSpentResponse> {
  return api.get<TimeSpentResponse>("/analytics/time-spent", {
    query: { from: params?.from, to: params?.to, group_by: params?.group_by },
  });
}
