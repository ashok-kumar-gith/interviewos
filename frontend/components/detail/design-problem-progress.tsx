"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, RotateCcw, Trash2 } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert } from "@/components/ui/alert";
import { SegmentedRating } from "@/components/ui/segmented-rating";
import {
  getDesignProblemProgress,
  saveDesignProblemProgress,
  deleteDesignProblemProgress,
  type DesignProblemProgress,
} from "@/lib/api/designproblems";
import { ApiError } from "@/lib/api/client";

/**
 * Progress card for an HLD design problem — mark it done and rate confidence,
 * mirroring how DSA/LLD problems are tracked. Surfaced on the detail page so the
 * design problem feeds the user's progress (and, in time, readiness).
 */
export function DesignProblemProgressCard({ id }: { id: string }) {
  const queryClient = useQueryClient();
  const key = ["design-problem-progress", id];
  const [confidence, setConfidence] = React.useState<number | undefined>();
  const [error, setError] = React.useState<string | null>(null);

  const query = useQuery<DesignProblemProgress, unknown>({
    queryKey: key,
    queryFn: () => getDesignProblemProgress(id),
    // Absent progress comes back as not_started (200), so a 404 here is the
    // problem itself missing — don't retry that.
    retry: (count, err) => !(err instanceof ApiError && err.status === 404) && count < 1,
  });

  React.useEffect(() => {
    if (query.data?.confidence != null) setConfidence(query.data.confidence);
  }, [query.data?.confidence]);

  const save = useMutation({
    mutationFn: (status: "completed" | "in_progress") =>
      saveDesignProblemProgress(id, { status, confidence: confidence ?? null }),
    onMutate: () => setError(null),
    onSuccess: (data) => {
      queryClient.setQueryData(key, data);
      void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    },
    onError: () => setError("Couldn't save your progress. Try again."),
  });

  const clear = useMutation({
    mutationFn: () => deleteDesignProblemProgress(id),
    onMutate: () => setError(null),
    onSuccess: () => {
      setConfidence(undefined);
      queryClient.setQueryData<DesignProblemProgress>(key, {
        design_problem_id: id,
        status: "not_started",
        attempts: 0,
        time_spent_minutes: 0,
      });
      void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    },
    onError: () => setError("Couldn't clear your progress. Try again."),
  });

  const status = query.data?.status ?? "not_started";
  const done = status === "completed";
  const started = status !== "not_started";
  const busy = save.isPending || clear.isPending;

  return (
    <Card className="space-y-4 p-5">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <CheckCircle2
            className={done ? "size-5 text-success" : "size-5 text-muted-foreground"}
            aria-hidden
          />
          <div>
            <p className="text-h3 font-semibold">Your progress</p>
            <p className="text-sm text-muted-foreground">
              {done
                ? "Completed — revisit anytime to refresh your confidence."
                : "Mark this design problem done once you can walk through it end to end."}
            </p>
          </div>
        </div>
        {done && (
          <span className="rounded-full bg-success/10 px-2.5 py-1 text-xs font-medium text-success">
            Done
          </span>
        )}
      </div>

      <div className="space-y-1.5">
        <span className="text-2xs uppercase tracking-wide text-muted-foreground">
          Confidence
        </span>
        <SegmentedRating
          name={`dp-confidence-${id}`}
          value={confidence}
          onChange={setConfidence}
          ariaLabel="Your confidence on this design problem (1–5)"
        />
      </div>

      {error && <Alert variant="danger">{error}</Alert>}

      <div className="flex flex-wrap gap-2">
        <Button onClick={() => save.mutate("completed")} loading={save.isPending} disabled={!confidence || busy}>
          <CheckCircle2 aria-hidden /> {done ? "Update" : "Mark complete"}
        </Button>
        {done && (
          <Button variant="outline" onClick={() => save.mutate("in_progress")} loading={save.isPending} disabled={busy}>
            <RotateCcw aria-hidden /> Revisit
          </Button>
        )}
        {started && (
          <Button variant="ghost" onClick={() => clear.mutate()} loading={clear.isPending} disabled={busy}>
            <Trash2 aria-hidden /> Clear progress
          </Button>
        )}
      </div>
    </Card>
  );
}
