"use client";

import * as React from "react";
import { Check, Loader2, X } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { SegmentedRating } from "@/components/ui/segmented-rating";
import { DifficultyPill } from "@/components/ui/difficulty-pill";
import { KindIcon, kindLabel } from "@/components/today/kind-icon";
import { TaskActionsMenu } from "@/components/today/task-actions-menu";
import type { PlanTask } from "@/lib/api/curriculum";
import type { ConfidenceLevel } from "@/lib/api/types";
import { pillarKey, pillarLabel } from "@/lib/pillar-meta";
import { cn } from "@/lib/utils";

export interface TodayTaskItemProps {
  task: PlanTask;
  onComplete: (input: { confidence: ConfidenceLevel; timeSpentMinutes: number }) => void;
  onSkip: () => void;
  onReschedule: (toDate: string) => void;
  onReopen?: () => void;
  onViewDetail: () => void;
  completing?: boolean;
  skipping?: boolean;
  rescheduling?: boolean;
  reopening?: boolean;
}

export function TodayTaskItem({
  task,
  onComplete,
  onSkip,
  onReschedule,
  onReopen,
  onViewDetail,
  completing = false,
  skipping = false,
  rescheduling = false,
  reopening = false,
}: TodayTaskItemProps) {
  const [expanded, setExpanded] = React.useState(false);
  const [confidence, setConfidence] = React.useState<number | undefined>(
    task.confidence ?? undefined,
  );
  const [minutes, setMinutes] = React.useState<string>(
    task.estimated_minutes ? String(task.estimated_minutes) : "",
  );

  const pkey = pillarKey(task.pillar_type);
  const done = task.status === "completed";
  const skipped = task.status === "skipped";
  const rescheduled = task.status === "rescheduled";
  const accent = `hsl(var(--pillar-${pkey}))`;

  function submit() {
    if (!confidence) return;
    onComplete({
      confidence: confidence as ConfidenceLevel,
      timeSpentMinutes: minutes ? Math.max(0, parseInt(minutes, 10) || 0) : task.estimated_minutes ?? 0,
    });
    setExpanded(false);
  }

  return (
    <Card
      className={cn(
        "p-4 transition-colors",
        done && "opacity-60",
        skipped && "opacity-50",
      )}
    >
      <div className="flex items-start gap-3">
        <button
          type="button"
          aria-label={done ? "Completed" : "Complete task"}
          aria-pressed={done}
          disabled={done || skipped || completing}
          onClick={() => (done ? undefined : setExpanded((v) => !v))}
          className={cn(
            "mt-0.5 grid size-5 shrink-0 place-items-center rounded-md border transition-colors",
            done
              ? "border-success bg-success text-success-foreground"
              : "border-border hover:border-foreground",
          )}
        >
          {completing ? (
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
          ) : done ? (
            <Check className="size-3.5" aria-hidden />
          ) : null}
        </button>

        <span
          className="mt-0.5 shrink-0 [&_svg]:size-4"
          style={{ color: accent }}
          title={kindLabel(task.kind)}
        >
          <KindIcon kind={task.kind} />
        </span>

        <div className="min-w-0 flex-1">
          <button
            type="button"
            onClick={onViewDetail}
            className={cn(
              "block text-left text-sm font-medium leading-tight hover:underline",
              (done || skipped) && "line-through",
            )}
          >
            {task.title}
          </button>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
            <Badge variant="outline" size="sm" style={{ borderColor: accent, color: accent }}>
              {pillarLabel(task.pillar_type)}
            </Badge>
            {task.estimated_minutes ? <span>{task.estimated_minutes} min</span> : null}
            {task.difficulty ? <DifficultyPill difficulty={task.difficulty} /> : null}
            {task.kind === "revise" && (
              <Badge variant="info" size="sm">
                Revise
              </Badge>
            )}
            {rescheduled && (
              <Badge variant="warning" size="sm">
                Rescheduled
              </Badge>
            )}
            {skipped && (
              <Badge variant="secondary" size="sm">
                Skipped
              </Badge>
            )}
            {done && task.confidence ? <span>Confidence {task.confidence}/5</span> : null}
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-1">
          <TaskActionsMenu
            triggerLabel={`Actions for ${task.title}`}
            onViewDetail={onViewDetail}
            onReschedule={onReschedule}
            onSkip={onSkip}
            onReopen={onReopen}
            rescheduling={rescheduling}
            skipping={skipping}
            reopening={reopening}
            showSkip={!done && !skipped}
            showReopen={(done || skipped) && !!onReopen}
          />
        </div>
      </div>

      {expanded && !done && !skipped && (
        <div className="mt-4 space-y-3 border-t border-border pt-4">
          <div className="space-y-2">
            <Label className="text-xs">How confident do you feel? (1–5)</Label>
            <SegmentedRating
              name={`confidence-${task.id}`}
              ariaLabel={`Confidence for ${task.title}, 1 to 5`}
              pillar={pkey}
              value={confidence}
              onChange={setConfidence}
            />
          </div>
          <div className="flex items-end gap-3">
            <div className="w-32">
              <Label htmlFor={`time-${task.id}`} className="text-xs">
                Time spent (min)
              </Label>
              <Input
                id={`time-${task.id}`}
                type="number"
                inputMode="numeric"
                min={0}
                value={minutes}
                onChange={(e) => setMinutes(e.target.value)}
                placeholder={task.estimated_minutes ? String(task.estimated_minutes) : "0"}
              />
            </div>
            <div className="flex items-center gap-2">
              <Button onClick={submit} disabled={!confidence} loading={completing}>
                {!completing && <Check aria-hidden />} Complete
              </Button>
              <Button variant="ghost" onClick={() => setExpanded(false)}>
                <X aria-hidden /> Cancel
              </Button>
            </div>
          </div>
        </div>
      )}
    </Card>
  );
}
