"use client";

import * as React from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, CalendarClock, SkipForward } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { KindIcon } from "@/components/today/kind-icon";
import { TaskDetailDialog } from "@/components/today/task-detail-dialog";
import { rescheduleTask, skipTask, type PlanTask } from "@/lib/api/curriculum";
import { useOverdueTasks } from "@/lib/use-overdue-tasks";
import { pillarKey, pillarLabel } from "@/lib/pillar-meta";

function formatShort(iso: string): string {
  const d = new Date(`${iso}T00:00:00`);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

/**
 * Carry-over surface: lists incomplete tasks scheduled before today with one-tap
 * "Move to today" (reschedule to today's date) and "Skip" actions. Invalidates
 * today/roadmap/overdue/dashboard queries after each action.
 */
export function OverdueSection() {
  const queryClient = useQueryClient();
  const { overdue, isLoading, hasRoadmap, today } = useOverdueTasks();
  const [error, setError] = React.useState<string | null>(null);
  const [detailTask, setDetailTask] = React.useState<PlanTask | null>(null);

  function invalidateAll() {
    void queryClient.invalidateQueries({ queryKey: ["overdue"] });
    void queryClient.invalidateQueries({ queryKey: ["roadmap"] });
    void queryClient.invalidateQueries({ queryKey: ["today"] });
    void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
  }

  const moveMutation = useMutation({
    mutationFn: ({ taskId }: { taskId: string }) =>
      rescheduleTask(taskId, { to_date: today }),
    onMutate: () => setError(null),
    onError: () => setError("Couldn't move that task. Try again."),
    onSettled: invalidateAll,
  });

  const skipMutation = useMutation({
    mutationFn: ({ taskId }: { taskId: string }) => skipTask(taskId),
    onMutate: () => setError(null),
    onError: () => setError("Couldn't skip that task. Try again."),
    onSettled: invalidateAll,
  });

  if (isLoading || !hasRoadmap || overdue.length === 0) return null;

  const busyId =
    (moveMutation.isPending && moveMutation.variables?.taskId) ||
    (skipMutation.isPending && skipMutation.variables?.taskId) ||
    null;

  return (
    <section aria-labelledby="overdue-heading" className="space-y-3">
      <div className="flex items-center gap-2">
        <AlertTriangle className="size-4 text-warning" aria-hidden />
        <h2 id="overdue-heading" className="text-h3 font-semibold">
          Overdue
        </h2>
        <Badge variant="warning" size="sm">
          {overdue.length}
        </Badge>
      </div>
      <p className="text-sm text-muted-foreground">
        These were scheduled earlier and aren&apos;t done yet. Move them to today or skip.
      </p>

      {error && <Alert variant="danger">{error}</Alert>}

      <ul className="space-y-2">
        {overdue.map(({ task, date }) => {
          const accent = `hsl(var(--pillar-${pillarKey(task.pillar_type)}))`;
          const busy = busyId === task.id;
          return (
            <li key={task.id}>
              <Card className="border-warning/40 p-3">
                <div className="flex items-center gap-3">
                  <span className="shrink-0 [&_svg]:size-4" style={{ color: accent }}>
                    <KindIcon kind={task.kind} />
                  </span>
                  <button
                    type="button"
                    onClick={() => setDetailTask(task)}
                    className="min-w-0 flex-1 text-left"
                  >
                    <p className="truncate text-sm font-medium hover:underline">{task.title}</p>
                    <div className="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                      <Badge
                        variant="outline"
                        size="sm"
                        style={{ borderColor: accent, color: accent }}
                      >
                        {pillarLabel(task.pillar_type)}
                      </Badge>
                      <span>Due {formatShort(date)}</span>
                      {task.estimated_minutes ? <span>{task.estimated_minutes} min</span> : null}
                    </div>
                  </button>
                  <div className="flex shrink-0 items-center gap-1.5">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => moveMutation.mutate({ taskId: task.id })}
                      loading={busy && moveMutation.isPending}
                      disabled={busy}
                    >
                      <CalendarClock aria-hidden />
                      Move to today
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={`Skip ${task.title}`}
                      onClick={() => skipMutation.mutate({ taskId: task.id })}
                      loading={busy && skipMutation.isPending}
                      disabled={busy}
                    >
                      {!(busy && skipMutation.isPending) && <SkipForward aria-hidden />}
                    </Button>
                  </div>
                </div>
              </Card>
            </li>
          );
        })}
      </ul>

      <TaskDetailDialog
        task={detailTask}
        open={detailTask !== null}
        onClose={() => setDetailTask(null)}
      />
    </section>
  );
}
