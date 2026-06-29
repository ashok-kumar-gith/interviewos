"use client";

import * as React from "react";
import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryResult,
} from "@tanstack/react-query";
import { CircleDot, RefreshCw, Rocket } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { DifficultyPill } from "@/components/ui/difficulty-pill";
import { KindIcon } from "@/components/today/kind-icon";
import { TaskActionsMenu } from "@/components/today/task-actions-menu";
import { TaskDetailDialog } from "@/components/today/task-detail-dialog";
import {
  getActiveRoadmap,
  getRoadmapWeek,
  rescheduleTask,
  skipTask,
  type PlanTask,
  type Roadmap,
  type RoadmapWeek,
} from "@/lib/api/curriculum";
import { ApiError } from "@/lib/api/client";
import { pillarKey, pillarLabel } from "@/lib/pillar-meta";
import { cn } from "@/lib/utils";

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

function formatRange(start: string, end: string): string {
  const fmt = (iso: string) => {
    const d = new Date(`${iso}T00:00:00`);
    return Number.isNaN(d.getTime())
      ? iso
      : d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  };
  return `${fmt(start)} – ${fmt(end)}`;
}

export default function RoadmapPage() {
  const queryClient = useQueryClient();
  const [selectedWeek, setSelectedWeek] = React.useState<number | null>(null);
  const [detailTask, setDetailTask] = React.useState<PlanTask | null>(null);
  const [actionError, setActionError] = React.useState<string | null>(null);

  const roadmapQuery = useQuery<Roadmap, unknown>({
    queryKey: ["roadmap", "active"],
    queryFn: getActiveRoadmap,
    retry: (count, error) => !(error instanceof ApiError && error.status === 404) && count < 1,
  });

  const roadmap = roadmapQuery.data;
  // Default the selected week to the one covering today, else week 1.
  const defaultWeek = React.useMemo(() => {
    if (!roadmap?.weeks?.length) return 1;
    const t = todayISO();
    const current = roadmap.weeks.find((w) => w.start_date <= t && t <= w.end_date);
    return current?.week_number ?? roadmap.weeks[0].week_number;
  }, [roadmap]);

  const weekNumber = selectedWeek ?? defaultWeek;

  const weekQuery = useQuery<RoadmapWeek, unknown>({
    queryKey: ["roadmap", roadmap?.id, "week", weekNumber],
    queryFn: () => getRoadmapWeek(roadmap!.id, weekNumber),
    enabled: !!roadmap?.id,
  });

  function invalidateTaskQueries() {
    void queryClient.invalidateQueries({ queryKey: ["roadmap"] });
    void queryClient.invalidateQueries({ queryKey: ["today"] });
    void queryClient.invalidateQueries({ queryKey: ["overdue"] });
    void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
  }

  const rescheduleMutation = useMutation({
    mutationFn: ({ taskId, toDate }: { taskId: string; toDate: string }) =>
      rescheduleTask(taskId, { to_date: toDate }),
    onMutate: () => setActionError(null),
    onError: () => setActionError("Couldn't reschedule that task. Try again."),
    onSettled: invalidateTaskQueries,
  });

  const skipMutation = useMutation({
    mutationFn: ({ taskId }: { taskId: string }) => skipTask(taskId),
    onMutate: () => setActionError(null),
    onError: () => setActionError("Couldn't skip that task. Try again."),
    onSettled: invalidateTaskQueries,
  });

  if (roadmapQuery.isLoading) return <RoadmapSkeleton />;

  const notFound = roadmapQuery.error instanceof ApiError && roadmapQuery.error.status === 404;
  if (notFound) {
    return (
      <Page>
        <EmptyState
          icon={Rocket}
          title="No active roadmap"
          description="Generate a personalized roadmap and your week-by-week plan will appear here."
          actionLabel="Start intake"
          actionHref="/intake"
        />
      </Page>
    );
  }

  if (roadmapQuery.isError || !roadmap) {
    return (
      <Page>
        <Alert variant="danger" title="Couldn't load your roadmap">
          Something went wrong. Try again.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => roadmapQuery.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      </Page>
    );
  }

  const weeks = roadmap.weeks ?? [];

  return (
    <Page subtitle={`${roadmap.total_weeks}-week plan · ${formatRange(roadmap.start_date, roadmap.end_date)}`}>
      {/* Week selector */}
      <div className="flex flex-wrap gap-2" role="tablist" aria-label="Roadmap weeks">
        {(weeks.length
          ? weeks.map((w) => w.week_number)
          : Array.from({ length: roadmap.total_weeks }, (_, i) => i + 1)
        ).map((n) => (
          <button
            key={n}
            type="button"
            role="tab"
            aria-selected={n === weekNumber}
            onClick={() => setSelectedWeek(n)}
            className={cn(
              "rounded-md border px-3 py-1.5 text-sm font-medium transition-colors",
              n === weekNumber
                ? "border-primary bg-primary text-primary-foreground"
                : "border-border bg-background text-muted-foreground hover:bg-muted",
            )}
          >
            Week {n}
          </button>
        ))}
      </div>

      {actionError && <Alert variant="danger">{actionError}</Alert>}

      <WeekPanel
        query={weekQuery}
        weekNumber={weekNumber}
        today={todayISO()}
        onViewDetail={setDetailTask}
        onReschedule={(taskId, toDate) => rescheduleMutation.mutate({ taskId, toDate })}
        onSkip={(taskId) => skipMutation.mutate({ taskId })}
        reschedulingId={
          rescheduleMutation.isPending ? rescheduleMutation.variables?.taskId ?? null : null
        }
        skippingId={skipMutation.isPending ? skipMutation.variables?.taskId ?? null : null}
      />

      <TaskDetailDialog
        task={detailTask}
        open={detailTask !== null}
        onClose={() => setDetailTask(null)}
      />
    </Page>
  );
}

function WeekPanel({
  query,
  weekNumber,
  today,
  onViewDetail,
  onReschedule,
  onSkip,
  reschedulingId,
  skippingId,
}: {
  query: UseQueryResult<RoadmapWeek, unknown>;
  weekNumber: number;
  today: string;
  onViewDetail: (task: PlanTask) => void;
  onReschedule: (taskId: string, toDate: string) => void;
  onSkip: (taskId: string) => void;
  reschedulingId: string | null;
  skippingId: string | null;
}) {
  if (query.isLoading || query.isFetching) {
    return (
      <div className="space-y-3" aria-busy>
        <Skeleton className="h-6 w-48" />
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-20" />
        ))}
      </div>
    );
  }

  if (query.isError || !query.data) {
    return (
      <Alert variant="danger">Couldn&apos;t load week {weekNumber}. Try another week.</Alert>
    );
  }

  const week = query.data;
  const days = week.days ?? [];

  return (
    <section className="space-y-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h2 className="text-h3 font-semibold">
            Week {week.week_number}
            {week.theme ? ` · ${week.theme}` : ""}
          </h2>
          <p className="text-sm text-muted-foreground">
            {formatRange(week.start_date, week.end_date)}
            {week.planned_hours ? ` · ~${week.planned_hours}h planned` : ""}
          </p>
        </div>
        {week.focus_pillars?.length ? (
          <div className="flex flex-wrap gap-1.5">
            {week.focus_pillars.map((p) => {
              const accent = `hsl(var(--pillar-${pillarKey(p)}))`;
              return (
                <Badge key={p} variant="outline" size="sm" style={{ borderColor: accent, color: accent }}>
                  {pillarLabel(p)}
                </Badge>
              );
            })}
          </div>
        ) : null}
      </div>

      {days.length === 0 ? (
        <Alert variant="info">No days scheduled for this week yet.</Alert>
      ) : (
        <ol className="space-y-3 border-l border-border pl-5">
          {days.map((day) => {
            const isToday = day.date === today;
            const isPast = day.date < today;
            const tasks = day.tasks ?? [];
            const done = tasks.filter((t) => t.status === "completed").length;
            return (
              <li key={day.id} className="relative" aria-current={isToday ? "date" : undefined}>
                <span
                  aria-hidden
                  className={cn(
                    "absolute -left-[27px] top-1 grid size-4 place-items-center rounded-full border-2 bg-background",
                    day.completed_minutes && day.planned_minutes && day.completed_minutes >= day.planned_minutes
                      ? "border-success bg-success"
                      : isToday
                        ? "border-primary"
                        : "border-border",
                  )}
                >
                  {isToday && <CircleDot className="size-3 text-primary" />}
                </span>
                <Card className="p-4">
                  <div className="flex items-baseline justify-between gap-2">
                    <p className="text-sm font-medium">
                      {new Date(`${day.date}T00:00:00`).toLocaleDateString(undefined, {
                        weekday: "short",
                        month: "short",
                        day: "numeric",
                      })}
                      {isToday && <span className="ml-2 text-xs text-primary">Today</span>}
                    </p>
                    <span className="text-xs text-muted-foreground">
                      {day.is_rest_day
                        ? "Rest day"
                        : tasks.length
                          ? `${done}/${tasks.length} done`
                          : "No tasks"}
                    </span>
                  </div>
                  {tasks.length > 0 && (
                    <ul className="mt-3 space-y-1">
                      {tasks.map((t) => {
                        const accent = `hsl(var(--pillar-${pillarKey(t.pillar_type)}))`;
                        const done = t.status === "completed";
                        const incomplete = t.status === "pending" || t.status === "in_progress";
                        const isOverdue = isPast && incomplete;
                        return (
                          <li key={t.id} className="flex items-center gap-2.5 text-sm">
                            <span className="shrink-0 [&_svg]:size-4" style={{ color: accent }}>
                              <KindIcon kind={t.kind} />
                            </span>
                            <button
                              type="button"
                              onClick={() => onViewDetail(t)}
                              className={cn(
                                "min-w-0 flex-1 truncate text-left hover:underline",
                                done && "text-muted-foreground line-through",
                              )}
                            >
                              {t.title}
                            </button>
                            {isOverdue && (
                              <Badge variant="warning" size="sm" className="shrink-0">
                                Overdue
                              </Badge>
                            )}
                            {t.status === "rescheduled" && (
                              <Badge variant="warning" size="sm" className="shrink-0">
                                Moved
                              </Badge>
                            )}
                            {t.difficulty ? <DifficultyPill difficulty={t.difficulty} /> : null}
                            {t.estimated_minutes ? (
                              <span className="shrink-0 text-xs text-muted-foreground">
                                {t.estimated_minutes}m
                              </span>
                            ) : null}
                            {incomplete && (
                              <span className="shrink-0">
                                <TaskActionsMenu
                                  triggerLabel={`Actions for ${t.title}`}
                                  onViewDetail={() => onViewDetail(t)}
                                  onReschedule={(toDate) => onReschedule(t.id, toDate)}
                                  onSkip={() => onSkip(t.id)}
                                  rescheduling={reschedulingId === t.id}
                                  skipping={skippingId === t.id}
                                />
                              </span>
                            )}
                          </li>
                        );
                      })}
                    </ul>
                  )}
                </Card>
              </li>
            );
          })}
        </ol>
      )}
    </section>
  );
}

function Page({
  children,
  subtitle,
}: {
  children: React.ReactNode;
  subtitle?: string;
}) {
  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-h1">Roadmap</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {subtitle ?? "Your week-by-week prep plan"}
        </p>
      </header>
      {children}
    </div>
  );
}

function RoadmapSkeleton() {
  return (
    <div className="space-y-6" aria-busy>
      <span className="sr-only" role="status">
        Loading roadmap
      </span>
      <header className="space-y-2">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-4 w-64" />
      </header>
      <div className="flex gap-2">
        {[0, 1, 2, 3, 4, 5].map((i) => (
          <Skeleton key={i} className="h-9 w-20" />
        ))}
      </div>
      <div className="space-y-3">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-24" />
        ))}
      </div>
    </div>
  );
}
