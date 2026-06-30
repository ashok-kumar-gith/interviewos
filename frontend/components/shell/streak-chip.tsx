"use client";

import { useQuery } from "@tanstack/react-query";
import { Flame } from "lucide-react";

import { getDashboard } from "@/lib/api/dashboard";
import { useAuthStore } from "@/lib/store/auth";

/**
 * Topbar streak chip showing the user's REAL current study streak (was a
 * hardcoded "12d" placeholder). Reuses the ["dashboard"] query so it shares the
 * dashboard page's cache — no extra request when the dashboard is open. Renders
 * nothing until authenticated or when the streak is 0, so a new user doesn't see
 * a meaningless "0d".
 */
export function StreakChip() {
  const authed = useAuthStore((s) => s.accessToken !== null);

  const { data } = useQuery({
    queryKey: ["dashboard"],
    queryFn: getDashboard,
    enabled: authed,
    staleTime: 60_000,
    // The dashboard returns an empty aggregate (404-like) for users with no
    // roadmap; treat any failure as "no streak" rather than surfacing an error.
    retry: false,
  });

  const current = data?.study_streak?.current ?? 0;
  if (!authed || current <= 0) return null;

  return (
    <span
      className="flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-1 text-xs font-medium tabular-nums"
      aria-label={`${current} day study streak`}
      title={`${current}-day study streak`}
    >
      <Flame className="size-3.5 text-warning" aria-hidden />
      {current}d
    </span>
  );
}
