"use client";

import { useQuery } from "@tanstack/react-query";

import {
  getActiveRoadmap,
  getRoadmapWeek,
  type PlanTask,
  type Roadmap,
} from "@/lib/api/curriculum";
import { ApiError } from "@/lib/api/client";

export interface OverdueTask {
  task: PlanTask;
  date: string;
}

function todayISO(): string {
  const d = new Date();
  const off = d.getTimezoneOffset();
  return new Date(d.getTime() - off * 60_000).toISOString().slice(0, 10);
}

const INCOMPLETE = new Set(["pending", "in_progress"]);

/**
 * Derives overdue tasks by scanning the active roadmap's past plan-days for
 * incomplete tasks (status pending/in_progress on a date before today). Fetches
 * the active roadmap, then the weeks that overlap past dates, and flattens.
 * Returns a stable query object so callers can show loading/error states.
 */
export function useOverdueTasks() {
  const today = todayISO();

  const roadmapQuery = useQuery<Roadmap, unknown>({
    queryKey: ["roadmap", "active"],
    queryFn: getActiveRoadmap,
    retry: (count, error) => !(error instanceof ApiError && error.status === 404) && count < 1,
  });

  const roadmap = roadmapQuery.data;
  // Week numbers that start on/before today — these may contain past days.
  const pastWeekNumbers = (roadmap?.weeks ?? [])
    .filter((w) => w.start_date <= today)
    .map((w) => w.week_number);

  const overdueQuery = useQuery<OverdueTask[], unknown>({
    queryKey: ["overdue", roadmap?.id, today],
    enabled: !!roadmap?.id && pastWeekNumbers.length > 0,
    queryFn: async () => {
      const weeks = await Promise.all(
        pastWeekNumbers.map((n) => getRoadmapWeek(roadmap!.id, n)),
      );
      const result: OverdueTask[] = [];
      for (const week of weeks) {
        for (const day of week.days ?? []) {
          if (day.is_rest_day || day.date >= today) continue;
          for (const task of day.tasks ?? []) {
            if (INCOMPLETE.has(task.status)) result.push({ task, date: day.date });
          }
        }
      }
      result.sort((a, b) => (a.date < b.date ? -1 : a.date > b.date ? 1 : 0));
      return result;
    },
  });

  return {
    overdue: overdueQuery.data ?? [],
    isLoading: roadmapQuery.isLoading || overdueQuery.isLoading,
    isFetching: overdueQuery.isFetching,
    hasRoadmap: !!roadmap?.id,
    today,
  };
}
