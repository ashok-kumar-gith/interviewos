"use client";

import * as React from "react";
import Link from "next/link";
import { ExternalLink, Target } from "lucide-react";

import { Dialog } from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Markdown } from "@/components/ui/markdown";
import { DifficultyPill } from "@/components/ui/difficulty-pill";
import { KindIcon, kindLabel } from "@/components/today/kind-icon";
import type { PlanTask } from "@/lib/api/curriculum";
import type { TaskStatus } from "@/lib/api/types";
import { pillarKey, pillarLabel } from "@/lib/pillar-meta";
import { taskItemHref, itemTypeLabel } from "@/lib/task-href";

const STATUS_LABEL: Record<TaskStatus, string> = {
  pending: "Pending",
  in_progress: "In progress",
  completed: "Completed",
  skipped: "Skipped",
  rescheduled: "Rescheduled",
};

const STATUS_VARIANT: Record<TaskStatus, "secondary" | "success" | "warning" | "info"> = {
  pending: "secondary",
  in_progress: "info",
  completed: "success",
  skipped: "secondary",
  rescheduled: "warning",
};

export interface TaskDetailDialogProps {
  task: PlanTask | null;
  open: boolean;
  onClose: () => void;
}

/**
 * Read-only detail view for a plan task: kind, pillar, objectives, description,
 * estimate, priority, difficulty, completion notes, and a link to the underlying
 * content (topic/problem/design/lld) when one exists. Opens as a modal Dialog.
 */
export function TaskDetailDialog({ task, open, onClose }: TaskDetailDialogProps) {
  if (!task) return null;

  const pkey = pillarKey(task.pillar_type);
  const accent = `hsl(var(--pillar-${pkey}))`;
  const href = taskItemHref(task.item_type, task.item_id);
  const objectives = task.objectives ?? [];

  return (
    <Dialog open={open} onClose={onClose} title={task.title} description={kindLabel(task.kind)}>
      <div className="space-y-5">
        <div className="flex flex-wrap items-center gap-1.5">
          <span
            className="inline-flex items-center gap-1.5 [&_svg]:size-4"
            style={{ color: accent }}
          >
            <KindIcon kind={task.kind} />
          </span>
          <Badge variant="outline" size="sm" style={{ borderColor: accent, color: accent }}>
            {pillarLabel(task.pillar_type)}
          </Badge>
          <Badge variant={STATUS_VARIANT[task.status]} size="sm">
            {STATUS_LABEL[task.status]}
          </Badge>
          {task.difficulty ? <DifficultyPill difficulty={task.difficulty} /> : null}
          {task.priority ? (
            <Badge variant="secondary" size="sm" className="capitalize">
              {task.priority} priority
            </Badge>
          ) : null}
          {task.estimated_minutes ? (
            <Badge variant="secondary" size="sm">
              ~{task.estimated_minutes} min
            </Badge>
          ) : null}
        </div>

        {task.description && task.description.trim() !== "" && (
          <section className="space-y-2">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              Description
            </h3>
            <Markdown content={task.description} />
          </section>
        )}

        {objectives.length > 0 && (
          <section className="space-y-2">
            <h3 className="flex items-center gap-1.5 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              <Target className="size-3.5" aria-hidden />
              Objectives
            </h3>
            <ul className="list-disc space-y-1 pl-5 text-sm marker:text-muted-foreground">
              {objectives.map((o, i) => (
                <li key={i}>{o}</li>
              ))}
            </ul>
          </section>
        )}

        {task.status === "completed" && task.completion_notes && (
          <section className="space-y-2">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              Your notes
            </h3>
            <Markdown content={task.completion_notes} />
            {task.confidence ? (
              <p className="text-xs text-muted-foreground">
                Confidence {task.confidence}/5
                {task.time_spent_minutes ? ` · ${task.time_spent_minutes} min spent` : ""}
              </p>
            ) : null}
          </section>
        )}

        {href && (
          <Link href={href} onClick={onClose} className="block">
            <Button variant="outline" className="w-full justify-center">
              <ExternalLink aria-hidden />
              Open {itemTypeLabel(task.item_type)}
            </Button>
          </Link>
        )}
      </div>
    </Dialog>
  );
}
