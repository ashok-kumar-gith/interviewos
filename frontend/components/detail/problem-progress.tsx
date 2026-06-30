"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Code2, Save, Trash2 } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert } from "@/components/ui/alert";
import { Textarea } from "@/components/ui/textarea";
import { SegmentedRating } from "@/components/ui/segmented-rating";
import { CodeEditor, type EditorLanguage } from "@/components/code/code-editor";
import {
  deleteProblemProgress,
  getProblemProgress,
  saveProblemProgress,
  type ProblemProgress,
} from "@/lib/api/dsaprogress";
import { ApiError } from "@/lib/api/client";
import { cn } from "@/lib/utils";

const LANGUAGES = ["python", "javascript", "typescript", "go", "java", "cpp", "c"] as const;

function formatWhen(iso?: string | null): string | null {
  if (!iso) return null;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return null;
  return d.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

/**
 * Per-problem progress + solution panel for the DSA detail page: mark solved,
 * rate confidence, and save the solution code (with language + a short note) so
 * the user has a record of which problem they solved, when, and how.
 */
export function ProblemProgressPanel({ problemId }: { problemId: string }) {
  const queryClient = useQueryClient();
  const key = ["problem-progress", problemId];

  const query = useQuery<ProblemProgress, unknown>({
    queryKey: key,
    queryFn: () => getProblemProgress(problemId),
    retry: (count, err) => !(err instanceof ApiError && err.status === 404) && count < 1,
  });

  const [confidence, setConfidence] = React.useState<number | undefined>();
  const [language, setLanguage] = React.useState<string>("python");
  const [code, setCode] = React.useState<string>("");
  const [solutionNote, setSolutionNote] = React.useState<string>("");
  const [error, setError] = React.useState<string | null>(null);

  // Hydrate the editor once the saved progress loads.
  const loaded = query.data;
  React.useEffect(() => {
    if (!loaded) return;
    if (loaded.confidence != null) setConfidence(loaded.confidence);
    if (loaded.solution_language) setLanguage(loaded.solution_language);
    if (loaded.solution_code != null) setCode(loaded.solution_code);
    if (loaded.solution_notes != null) setSolutionNote(loaded.solution_notes);
  }, [loaded]);

  const save = useMutation({
    mutationFn: (solved: boolean) =>
      saveProblemProgress(problemId, {
        solved,
        confidence: confidence ?? null,
        solution_code: code.trim() || null,
        solution_language: code.trim() ? language : null,
        solution_notes: solutionNote.trim() || null,
      }),
    onMutate: () => setError(null),
    onSuccess: (data) => {
      queryClient.setQueryData(key, data);
      void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
      void queryClient.invalidateQueries({ queryKey: ["problems-solved"] });
    },
    onError: () => setError("Couldn't save your progress. Try again."),
  });

  const clear = useMutation({
    mutationFn: () => deleteProblemProgress(problemId),
    onMutate: () => setError(null),
    onSuccess: () => {
      setConfidence(undefined);
      setCode("");
      setSolutionNote("");
      void queryClient.invalidateQueries({ queryKey: key });
      void queryClient.invalidateQueries({ queryKey: ["problems-solved"] });
    },
    onError: () => setError("Couldn't clear your progress. Try again."),
  });

  const solved = loaded?.solved ?? false;
  const solvedWhen = formatWhen(loaded?.solved_at);

  return (
    <Card className="space-y-4 p-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <CheckCircle2
            className={cn("size-5", solved ? "text-success" : "text-muted-foreground")}
            aria-hidden
          />
          <div>
            <p className="text-h3 font-semibold">Your solution</p>
            <p className="text-sm text-muted-foreground">
              {solved && solvedWhen ? `Solved ${solvedWhen}` : "Track when you solved it and save your code."}
            </p>
          </div>
        </div>
        {solved && (
          <span className="rounded-full bg-success/10 px-2.5 py-1 text-xs font-medium text-success">
            Solved
          </span>
        )}
      </div>

      <div className="space-y-1.5">
        <span className="text-2xs uppercase tracking-wide text-muted-foreground">Confidence</span>
        <SegmentedRating
          name={`problem-confidence-${problemId}`}
          value={confidence}
          onChange={setConfidence}
          ariaLabel="Your confidence on this problem (1–5)"
        />
      </div>

      <div className="space-y-1.5">
        <div className="flex items-center gap-2">
          <Code2 className="size-4 text-muted-foreground" aria-hidden />
          <label htmlFor={`lang-${problemId}`} className="text-2xs uppercase tracking-wide text-muted-foreground">
            Solution
          </label>
          <select
            id={`lang-${problemId}`}
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
            className="ml-auto rounded-md border border-border bg-background px-2 py-1 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            aria-label="Solution language"
          >
            {LANGUAGES.map((l) => (
              <option key={l} value={l}>
                {l}
              </option>
            ))}
          </select>
        </div>
        <CodeEditor
          language={language as EditorLanguage}
          value={code}
          onChange={setCode}
          rows={10}
          aria-label="Your solution code"
          placeholder="// Paste or write your solution here"
        />
      </div>

      <Textarea
        value={solutionNote}
        onChange={(e) => setSolutionNote(e.target.value)}
        rows={2}
        placeholder="Approach notes — the key insight, complexity, gotchas."
      />

      {error && <Alert variant="danger">{error}</Alert>}

      <div className="flex flex-wrap gap-2">
        <Button onClick={() => save.mutate(true)} loading={save.isPending && save.variables === true}>
          <CheckCircle2 aria-hidden /> {solved ? "Update solution" : "Mark solved & save"}
        </Button>
        <Button
          variant="outline"
          onClick={() => save.mutate(false)}
          loading={save.isPending && save.variables === false}
        >
          <Save aria-hidden /> Save without solving
        </Button>
        {(solved || loaded?.status !== "not_started") && (
          <Button variant="ghost" onClick={() => clear.mutate()} loading={clear.isPending}>
            <Trash2 aria-hidden /> Clear
          </Button>
        )}
      </div>
    </Card>
  );
}
