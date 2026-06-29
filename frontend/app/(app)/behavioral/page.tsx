"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  MessagesSquare,
  Pencil,
  Plus,
  RefreshCw,
  Sparkles,
  Trash2,
} from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { Dialog } from "@/components/ui/dialog";
import { StarStoryEditor } from "@/components/behavioral/star-story-editor";
import { ApiError } from "@/lib/api/client";
import {
  createStory,
  deleteStory,
  improveStory,
  listStories,
  themeLabel,
  updateStory,
  type BehavioralStory,
  type BehavioralStoryUpsert,
  type StoryImproveResponse,
  type StoryTheme,
} from "@/lib/api/behavioral";

const STORIES_KEY = ["behavioral-stories"] as const;

type EditorState =
  | { mode: "closed" }
  | { mode: "create" }
  | { mode: "edit"; story: BehavioralStory };

export default function BehavioralPage() {
  const queryClient = useQueryClient();
  const [editor, setEditor] = React.useState<EditorState>({ mode: "closed" });
  const [editorError, setEditorError] = React.useState<string | null>(null);
  const [improveFor, setImproveFor] = React.useState<BehavioralStory | null>(null);
  const [improveResult, setImproveResult] = React.useState<StoryImproveResponse | null>(null);
  const [improveError, setImproveError] = React.useState<string | null>(null);
  const [actionError, setActionError] = React.useState<string | null>(null);

  const query = useQuery<BehavioralStory[], unknown>({
    queryKey: STORIES_KEY,
    queryFn: () => listStories(),
  });

  function closeEditor() {
    setEditor({ mode: "closed" });
    setEditorError(null);
  }

  const saveMutation = useMutation({
    mutationFn: (payload: BehavioralStoryUpsert) =>
      editor.mode === "edit"
        ? updateStory(editor.story.id, payload)
        : createStory(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: STORIES_KEY });
      closeEditor();
    },
    onError: () => setEditorError("Couldn't save your story. Check your connection and retry."),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteStory(id),
    onMutate: () => setActionError(null),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: STORIES_KEY }),
    onError: () => setActionError("Couldn't delete that story. Try again."),
  });

  const improveMutation = useMutation({
    mutationFn: (id: string) => improveStory(id),
    onMutate: () => {
      setImproveError(null);
      setImproveResult(null);
    },
    onSuccess: (data) => {
      setImproveResult(data);
      void queryClient.invalidateQueries({ queryKey: STORIES_KEY });
    },
    onError: (err) =>
      setImproveError(
        err instanceof ApiError && err.status === 503
          ? "The AI coach is busy right now. Try again in a moment."
          : "Couldn't improve that story. Try again.",
      ),
  });

  if (query.isLoading) return <BehavioralSkeleton />;

  if (query.isError) {
    return (
      <Page onNew={() => setEditor({ mode: "create" })}>
        <Alert variant="danger" title="Couldn't load your stories">
          Something went wrong.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => query.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      </Page>
    );
  }

  const stories = query.data ?? [];
  const byTheme = groupByTheme(stories);
  const themes = Object.keys(byTheme) as StoryTheme[];

  return (
    <Page onNew={() => setEditor({ mode: "create" })}>
      {actionError && <Alert variant="danger">{actionError}</Alert>}

      {stories.length === 0 ? (
        <EmptyState
          icon={MessagesSquare}
          title="No stories yet"
          description="Capture your strongest moments as STAR stories — leadership, conflict, impact. The interview is easier when the stories are ready."
        >
          <Button className="mt-2" onClick={() => setEditor({ mode: "create" })}>
            <Plus aria-hidden /> New story
          </Button>
        </EmptyState>
      ) : (
        <div className="space-y-8">
          {themes.map((theme) => (
            <section key={theme} className="space-y-3">
              <h2 className="flex items-center gap-2 text-h3">
                <span
                  className="size-2 rounded-full"
                  style={{ backgroundColor: "hsl(var(--pillar-behavioral))" }}
                  aria-hidden
                />
                {themeLabel(theme)}
                <span className="text-sm font-normal text-muted-foreground">
                  {byTheme[theme].length}
                </span>
              </h2>
              <div className="grid gap-3 md:grid-cols-2">
                {byTheme[theme].map((story) => (
                  <StoryCard
                    key={story.id}
                    story={story}
                    onEdit={() => setEditor({ mode: "edit", story })}
                    onImprove={() => {
                      setImproveFor(story);
                      improveMutation.mutate(story.id);
                    }}
                    onDelete={() => deleteMutation.mutate(story.id)}
                    deleting={deleteMutation.isPending && deleteMutation.variables === story.id}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      )}

      <Dialog
        open={editor.mode !== "closed"}
        onClose={closeEditor}
        title={editor.mode === "edit" ? "Edit story" : "New STAR story"}
        description="Situation, Task, Action, Result — the structure interviewers expect."
      >
        {editor.mode !== "closed" && (
          <StarStoryEditor
            initial={editor.mode === "edit" ? editor.story : undefined}
            saving={saveMutation.isPending}
            error={editorError}
            onSubmit={(payload) => saveMutation.mutate(payload)}
            onImprove={
              editor.mode === "edit"
                ? () => {
                    setImproveFor(editor.story);
                    improveMutation.mutate(editor.story.id);
                  }
                : undefined
            }
            improving={improveMutation.isPending}
            onCancel={closeEditor}
          />
        )}
      </Dialog>

      <Dialog
        open={Boolean(improveFor)}
        onClose={() => {
          setImproveFor(null);
          setImproveResult(null);
          setImproveError(null);
        }}
        title="AI suggestions"
        description={improveFor ? improveFor.title : undefined}
      >
        {improveMutation.isPending ? (
          <div className="space-y-3" aria-busy>
            <span className="sr-only" role="status">
              Improving your story
            </span>
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        ) : improveError ? (
          <Alert variant="danger">{improveError}</Alert>
        ) : improveResult ? (
          <div className="space-y-4">
            {improveResult.strength_score !== undefined && (
              <div className="flex items-center gap-3">
                <span className="text-2xs uppercase text-muted-foreground">Story strength</span>
                <StrengthMeter score={improveResult.strength_score} />
              </div>
            )}
            {improveResult.used_fallback && (
              <Alert variant="warning">
                The AI coach was unavailable — these are heuristic suggestions.
              </Alert>
            )}
            {improveResult.suggestions.length > 0 ? (
              <ul className="space-y-2">
                {improveResult.suggestions.map((s, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm">
                    <Sparkles
                      className="mt-0.5 size-4 shrink-0"
                      style={{ color: "hsl(var(--pillar-behavioral))" }}
                      aria-hidden
                    />
                    <span>{s}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground">
                This story already reads well — no changes suggested.
              </p>
            )}
            {improveResult.improved && (
              <ImprovedSections improved={improveResult.improved} />
            )}
          </div>
        ) : null}
      </Dialog>
    </Page>
  );
}

function Page({ children, onNew }: { children: React.ReactNode; onNew: () => void }) {
  return (
    <div className="space-y-6">
      <header className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-h1">Behavioral</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Your STAR stories, organized by theme.
          </p>
        </div>
        <Button onClick={onNew}>
          <Plus aria-hidden /> New story
        </Button>
      </header>
      {children}
    </div>
  );
}

function StoryCard({
  story,
  onEdit,
  onImprove,
  onDelete,
  deleting,
}: {
  story: BehavioralStory;
  onEdit: () => void;
  onImprove: () => void;
  onDelete: () => void;
  deleting: boolean;
}) {
  const preview = story.situation || story.action || story.result || "No details yet.";
  return (
    <Card className="flex h-full flex-col">
      <CardContent className="flex flex-1 flex-col gap-3 p-5">
        <div className="flex items-start justify-between gap-2">
          <h3 className="font-medium leading-snug">{story.title}</h3>
          {story.strength_score != null && <StrengthBadge score={story.strength_score} />}
        </div>
        <p className="line-clamp-2 text-sm text-muted-foreground">{preview}</p>
        {story.tags && story.tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {story.tags.map((t) => (
              <Badge key={t} variant="outline" size="sm">
                {t}
              </Badge>
            ))}
          </div>
        )}
        <div className="mt-auto flex items-center gap-1 pt-1">
          <Button variant="ghost" size="sm" onClick={onEdit}>
            <Pencil aria-hidden /> Edit
          </Button>
          <Button variant="ghost" size="sm" onClick={onImprove}>
            <Sparkles aria-hidden /> Improve
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="ml-auto text-muted-foreground hover:text-danger"
            onClick={onDelete}
            loading={deleting}
            aria-label={`Delete story: ${story.title}`}
          >
            <Trash2 aria-hidden />
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function ImprovedSections({ improved }: { improved: NonNullable<StoryImproveResponse["improved"]> }) {
  const rows: [string, string | undefined][] = [
    ["Situation", improved.situation],
    ["Task", improved.task],
    ["Action", improved.action],
    ["Result", improved.result],
    ["Metrics", improved.metrics],
  ];
  const filled = rows.filter(([, v]) => v && v.trim());
  if (filled.length === 0) return null;
  return (
    <div className="space-y-3 rounded-lg border border-border bg-muted/30 p-4">
      <p className="text-2xs uppercase text-muted-foreground">Suggested rewrite</p>
      {filled.map(([label, v]) => (
        <div key={label}>
          <p className="text-xs font-medium text-foreground">{label}</p>
          <p className="text-sm text-muted-foreground">{v}</p>
        </div>
      ))}
    </div>
  );
}

function scoreTone(score: number): { variant: "danger" | "warning" | "success"; color: string } {
  if (score >= 75) return { variant: "success", color: "hsl(var(--success))" };
  if (score >= 50) return { variant: "warning", color: "hsl(var(--warning))" };
  return { variant: "danger", color: "hsl(var(--danger))" };
}

function StrengthMeter({ score }: { score: number }) {
  const pct = Math.max(0, Math.min(100, Math.round(score)));
  const { color } = scoreTone(pct);
  return (
    <div className="flex flex-1 items-center gap-2">
      <div
        className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted"
        role="progressbar"
        aria-valuenow={pct}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={`Story strength ${pct} out of 100`}
      >
        <div className="h-full rounded-full" style={{ width: `${pct}%`, backgroundColor: color }} />
      </div>
      <span className="text-sm font-semibold tabular-nums">{pct}</span>
    </div>
  );
}

function StrengthBadge({ score }: { score: number }) {
  const pct = Math.round(score);
  const { variant } = scoreTone(pct);
  return (
    <Badge variant={variant} size="sm" className="tabular-nums">
      {pct}
    </Badge>
  );
}

function groupByTheme(stories: BehavioralStory[]): Record<string, BehavioralStory[]> {
  const out: Record<string, BehavioralStory[]> = {};
  for (const s of stories) {
    (out[s.theme] ??= []).push(s);
  }
  return out;
}

function BehavioralSkeleton() {
  return (
    <div className="space-y-6" aria-busy>
      <span className="sr-only" role="status">
        Loading your stories
      </span>
      <header className="space-y-2">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-4 w-56" />
      </header>
      <div className="grid gap-3 md:grid-cols-2">
        {[0, 1, 2, 3].map((i) => (
          <Card key={i} className="p-5">
            <Skeleton className="h-5 w-2/3" />
            <Skeleton className="mt-3 h-4 w-full" />
            <Skeleton className="mt-2 h-4 w-3/4" />
          </Card>
        ))}
      </div>
    </div>
  );
}
