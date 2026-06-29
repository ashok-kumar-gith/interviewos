"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CalendarOff, CheckCircle2, Coffee, RefreshCw, Rocket } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { TodayTaskItem } from "@/components/today/today-task-item";
import { OverdueSection } from "@/components/today/overdue-section";
import { TaskDetailDialog } from "@/components/today/task-detail-dialog";
import {
  completeTask,
  getToday,
  rescheduleTask,
  skipTask,
  type PlanDay,
  type PlanTask,
} from "@/lib/api/curriculum";
import type { ConfidenceLevel } from "@/lib/api/types";
import { ApiError } from "@/lib/api/client";

const TODAY_KEY = ["today"] as const;

function formatDate(iso: string): string {
  const d = new Date(`${iso}T00:00:00`);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, {
    weekday: "long",
    month: "long",
    day: "numeric",
  });
}

export default function TodayPage() {
  const queryClient = useQueryClient();
  const [actionError, setActionError] = React.useState<string | null>(null);
  const [detailTask, setDetailTask] = React.useState<PlanTask | null>(null);

  const query = useQuery<PlanDay, unknown>({
    queryKey: TODAY_KEY,
    queryFn: getToday,
    retry: (count, error) => !(error instanceof ApiError && error.status === 404) && count < 1,
  });

  const completeMutation = useMutation({
    mutationFn: ({
      taskId,
      confidence,
      timeSpentMinutes,
    }: {
      taskId: string;
      confidence: ConfidenceLevel;
      timeSpentMinutes: number;
    }) => completeTask(taskId, { confidence, time_spent_minutes: timeSpentMinutes }),
    onMutate: async ({ taskId, confidence }) => {
      setActionError(null);
      await queryClient.cancelQueries({ queryKey: TODAY_KEY });
      const previous = queryClient.getQueryData<PlanDay>(TODAY_KEY);
      queryClient.setQueryData<PlanDay>(TODAY_KEY, (old) =>
        old ? patchTask(old, taskId, { status: "completed", confidence }) : old,
      );
      return { previous };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.previous) queryClient.setQueryData(TODAY_KEY, ctx.previous);
      setActionError("Couldn't complete that task. Try again.");
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: TODAY_KEY });
      void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    },
  });

  const skipMutation = useMutation({
    mutationFn: ({ taskId }: { taskId: string }) => skipTask(taskId),
    onMutate: async ({ taskId }) => {
      setActionError(null);
      await queryClient.cancelQueries({ queryKey: TODAY_KEY });
      const previous = queryClient.getQueryData<PlanDay>(TODAY_KEY);
      queryClient.setQueryData<PlanDay>(TODAY_KEY, (old) =>
        old ? patchTask(old, taskId, { status: "skipped" }) : old,
      );
      return { previous };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.previous) queryClient.setQueryData(TODAY_KEY, ctx.previous);
      setActionError("Couldn't skip that task. Try again.");
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: TODAY_KEY });
      void queryClient.invalidateQueries({ queryKey: ["overdue"] });
    },
  });

  const rescheduleMutation = useMutation({
    mutationFn: ({ taskId, toDate }: { taskId: string; toDate: string }) =>
      rescheduleTask(taskId, { to_date: toDate }),
    onMutate: async ({ taskId }) => {
      setActionError(null);
      await queryClient.cancelQueries({ queryKey: TODAY_KEY });
      const previous = queryClient.getQueryData<PlanDay>(TODAY_KEY);
      queryClient.setQueryData<PlanDay>(TODAY_KEY, (old) =>
        old ? patchTask(old, taskId, { status: "rescheduled" }) : old,
      );
      return { previous };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.previous) queryClient.setQueryData(TODAY_KEY, ctx.previous);
      setActionError("Couldn't reschedule that task. Try again.");
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: TODAY_KEY });
      void queryClient.invalidateQueries({ queryKey: ["roadmap"] });
      void queryClient.invalidateQueries({ queryKey: ["overdue"] });
      void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    },
  });

  if (query.isLoading) return <TodaySkeleton />;

  const notFound = query.error instanceof ApiError && query.error.status === 404;
  if (notFound) {
    return (
      <Page>
        <EmptyState
          icon={Rocket}
          title="No plan yet"
          description="Generate your roadmap and we'll build today's tasks for you."
          actionLabel="Start intake"
          actionHref="/intake"
        />
      </Page>
    );
  }

  if (query.isError) {
    return (
      <Page>
        <Alert variant="danger" title="Couldn't load today's plan">
          Something went wrong. Try again.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => query.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      </Page>
    );
  }

  const day = query.data!;
  const tasks = day.tasks ?? [];
  const completed = tasks.filter((t) => t.status === "completed").length;
  const progress = tasks.length ? Math.round((completed / tasks.length) * 100) : 0;

  const taskList = (
    <ol className="space-y-3">
      {tasks.map((task) => (
        <li key={task.id}>
          <TodayTaskItem
            task={task}
            completing={
              completeMutation.isPending && completeMutation.variables?.taskId === task.id
            }
            skipping={skipMutation.isPending && skipMutation.variables?.taskId === task.id}
            rescheduling={
              rescheduleMutation.isPending && rescheduleMutation.variables?.taskId === task.id
            }
            onComplete={({ confidence, timeSpentMinutes }) =>
              completeMutation.mutate({ taskId: task.id, confidence, timeSpentMinutes })
            }
            onSkip={() => skipMutation.mutate({ taskId: task.id })}
            onReschedule={(toDate) => rescheduleMutation.mutate({ taskId: task.id, toDate })}
            onViewDetail={() => setDetailTask(task)}
          />
        </li>
      ))}
    </ol>
  );

  return (
    <Page dateLabel={formatDate(day.date)} completed={completed} total={tasks.length}>
      {tasks.length > 0 && (
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted" aria-hidden>
          <div
            className="h-full rounded-full bg-primary transition-all"
            style={{ width: `${progress}%` }}
          />
        </div>
      )}

      {actionError && <Alert variant="danger">{actionError}</Alert>}

      <OverdueSection />

      {day.is_rest_day ? (
        <EmptyState
          icon={Coffee}
          title="Rest day"
          description="No tasks scheduled — recovery is part of the plan. See you tomorrow."
        />
      ) : tasks.length === 0 ? (
        <EmptyState
          icon={CalendarOff}
          title="Nothing scheduled today"
          description="There are no tasks for today. Check your roadmap for what's coming up."
          actionLabel="View roadmap"
          actionHref="/roadmap"
        />
      ) : completed === tasks.length ? (
        <>
          <Alert variant="success" title="All done for today">
            You completed every task. Nice work — your streak thanks you.
          </Alert>
          {taskList}
        </>
      ) : (
        taskList
      )}

      <TaskDetailDialog
        task={detailTask}
        open={detailTask !== null}
        onClose={() => setDetailTask(null)}
      />
    </Page>
  );
}

function Page({
  children,
  dateLabel,
  completed,
  total,
}: {
  children: React.ReactNode;
  dateLabel?: string;
  completed?: number;
  total?: number;
}) {
  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-h1">Today</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {dateLabel ?? "Your plan for today"}
          {total !== undefined && total > 0 ? ` · ${completed} of ${total} done` : ""}
        </p>
      </header>
      {children}
    </div>
  );
}

function TodaySkeleton() {
  return (
    <div className="space-y-6" aria-busy>
      <span className="sr-only" role="status">
        Loading today&apos;s plan
      </span>
      <header className="space-y-2">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-4 w-56" />
      </header>
      <Skeleton className="h-1.5 w-full" />
      <div className="space-y-3">
        {[0, 1, 2, 3].map((i) => (
          <Card key={i} className="flex items-start gap-3 p-4">
            <Skeleton className="size-5 rounded-md" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-2/3" />
              <Skeleton className="h-3 w-1/3" />
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}

/** Immutably patch a single task within a plan day (optimistic update). */
function patchTask(day: PlanDay, taskId: string, patch: Partial<PlanTask>): PlanDay {
  return {
    ...day,
    tasks: (day.tasks ?? []).map((t) => (t.id === taskId ? { ...t, ...patch } : t)),
  };
}
