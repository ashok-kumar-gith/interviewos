"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import {
  CalendarCheck,
  Flame,
  ListTodo,
  RefreshCw,
  Rocket,
} from "lucide-react";

import { ReadinessCard } from "@/components/dashboard/readiness-card";
import { PillarRadar } from "@/components/dashboard/pillar-radar";
import { StreakHeatmap } from "@/components/dashboard/streak-heatmap";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert } from "@/components/ui/alert";
import { EmptyState } from "@/components/ui/empty-state";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/button";
import { getDashboard, type DashboardResponse } from "@/lib/api/dashboard";
import { ApiError } from "@/lib/api/client";
import { pillarKey, pillarLabel } from "@/lib/pillar-meta";
import { cn } from "@/lib/utils";

function formatReadyDate(iso?: string | null): string | null {
  if (!iso) return null;
  const d = new Date(`${iso}T00:00:00`);
  if (Number.isNaN(d.getTime())) return null;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

export default function DashboardPage() {
  const query = useQuery<DashboardResponse, unknown>({
    queryKey: ["dashboard"],
    queryFn: getDashboard,
    retry: (count, error) => !(error instanceof ApiError && error.status === 404) && count < 1,
  });

  if (query.isLoading) {
    return <DashboardSkeleton />;
  }

  // No profile/roadmap yet → onboarding CTA. The backend returns 404 in some
  // cases, but GetDashboard also returns 200 with an all-empty aggregate before
  // a roadmap exists — treat that as "needs onboarding" too, otherwise a brand
  // new user lands on a zeroed dashboard with no way forward.
  const notFound = query.error instanceof ApiError && query.error.status === 404;
  const d = query.data;
  const emptyAggregate =
    !query.isError &&
    !!d &&
    (d.pillar_readiness?.length ?? 0) === 0 &&
    (d.today?.total_tasks ?? 0) === 0;
  if (notFound || emptyAggregate) {
    return (
      <div className="space-y-8">
        <header>
          <h1 className="text-h1">Dashboard</h1>
        </header>
        <EmptyState
          icon={Rocket}
          title="Generate your roadmap"
          description="Tell us where you are and we'll build a personalized, day-by-day prep plan. Your readiness, streak, and today's tasks will show up here."
          actionLabel="Start intake"
          actionHref="/intake"
        />
      </div>
    );
  }

  if (query.isError) {
    return (
      <div className="space-y-8">
        <header>
          <h1 className="text-h1">Dashboard</h1>
        </header>
        <Alert variant="danger" title="Couldn't load your dashboard">
          Something went wrong fetching your readiness. Try again.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => query.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      </div>
    );
  }

  const data = query.data!;
  const readyDate = formatReadyDate(data.estimated_readiness_date);
  const overall = Math.round(data.overall_readiness);
  const { today } = data;

  return (
    <div className="space-y-8">
      <header className="space-y-4">
        <div>
          <h1 className="text-h1">Dashboard</h1>
          <p className="mt-1 text-sm text-muted-foreground">You&apos;re {overall}% ready</p>
        </div>
        <Card
          className={cn(
            "flex items-center gap-3 p-4",
            readyDate ? "border-primary/40 bg-primary/5" : "border-border",
          )}
        >
          <span
            className={cn(
              "grid size-10 shrink-0 place-items-center rounded-full",
              readyDate ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground",
            )}
          >
            <CalendarCheck className="size-5" aria-hidden />
          </span>
          {readyDate ? (
            <div>
              <p className="text-2xs uppercase tracking-wide text-muted-foreground">
                Est. interview-ready
              </p>
              <p className="text-h3 font-bold tabular-nums">{readyDate}</p>
            </div>
          ) : (
            <div>
              <p className="text-sm font-medium">Est. interview-ready: not projected yet</p>
              <p className="text-sm text-muted-foreground">
                Keep completing tasks — we&apos;ll project a date as your progress accrues.
              </p>
            </div>
          )}
        </Card>
      </header>

      <section aria-labelledby="overall-readiness" className="grid gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-4">
          <h2 id="overall-readiness" className="sr-only">
            Overall readiness
          </h2>
          <ReadinessCard
            label="Overall"
            value={overall}
            takeaway={
              readyDate ? `On track — estimated ready by ${readyDate}.` : "Keep building momentum."
            }
            variant="overall"
          />

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <StatCard
              icon={Flame}
              label="Current streak"
              value={`${data.study_streak.current} ${data.study_streak.current === 1 ? "day" : "days"}`}
              hint={`Longest ${data.study_streak.longest}`}
            />
            <StatCard
              icon={ListTodo}
              label="Today"
              value={`${today.completed_tasks}/${today.total_tasks} done`}
              hint={`${today.remaining_hours.toFixed(1)}h remaining`}
            />
            <StatCard
              icon={CalendarCheck}
              label="Revision due"
              value={`${data.revision_due_count}`}
              hint={data.revision_due_count === 0 ? "All caught up" : "Items to recall"}
            />
          </div>
        </div>

        <PillarRadar
          data={data.pillar_readiness.map((p) => ({
            pillar: pillarLabel(p.pillar),
            readiness: Math.round(p.readiness),
          }))}
        />
      </section>

      <section aria-labelledby="today-cta">
        <Card className="flex flex-col items-start justify-between gap-3 p-5 sm:flex-row sm:items-center">
          <div>
            <h2 id="today-cta" className="text-h3 font-semibold">
              Today&apos;s plan
            </h2>
            <p className="mt-0.5 text-sm text-muted-foreground">
              {today.total_tasks === 0
                ? "No tasks scheduled for today."
                : `${today.total_tasks} tasks · ~${today.estimated_hours.toFixed(1)}h planned`}
            </p>
          </div>
          <Link href="/today" className={cn(buttonVariants())}>
            Go to Today
          </Link>
        </Card>
      </section>

      <section aria-labelledby="study-activity" className="space-y-4">
        <h2 id="study-activity" className="sr-only">
          Study activity
        </h2>
        <StreakHeatmap />
      </section>

      <section aria-labelledby="pillar-readiness" className="space-y-4">
        <h2 id="pillar-readiness" className="text-h3">
          Readiness by pillar
        </h2>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {data.pillar_readiness.map((p) => (
            <ReadinessCard
              key={p.pillar}
              label={pillarLabel(p.pillar)}
              value={Math.round(p.readiness)}
              takeaway={`Coverage ${Math.round(p.coverage * 100)}% · Confidence ${p.avg_confidence.toFixed(1)}/5`}
              pillar={pillarKey(p.pillar)}
              variant="pillar"
            />
          ))}
        </div>
      </section>
    </div>
  );
}

function StatCard({
  icon: Icon,
  label,
  value,
  hint,
}: {
  icon: typeof Flame;
  label: string;
  value: string;
  hint?: string;
}) {
  return (
    <Card className="p-4">
      <div className="flex items-center gap-2 text-muted-foreground">
        <Icon className="size-4" aria-hidden />
        <span className="text-2xs uppercase">{label}</span>
      </div>
      <p className="mt-1.5 text-h3 font-bold tabular-nums">{value}</p>
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </Card>
  );
}

function DashboardSkeleton() {
  return (
    <div className="space-y-8" aria-busy>
      <span className="sr-only" role="status">
        Loading dashboard
      </span>
      <header className="space-y-2">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-4 w-64" />
      </header>
      <div className="grid gap-4 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          <ReadinessCard label="Overall" value={0} variant="overall" loading />
          <div className="grid gap-4 sm:grid-cols-3">
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-24" />
            ))}
          </div>
        </div>
        <Skeleton className="h-64" />
      </div>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {[0, 1, 2, 3, 4, 5].map((i) => (
          <ReadinessCard key={i} label="" value={0} loading />
        ))}
      </div>
    </div>
  );
}
