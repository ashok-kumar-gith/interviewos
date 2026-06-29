"use client";

import * as React from "react";
import { Sparkles } from "lucide-react";

import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { SelectField } from "@/components/ui/select-field";
import { TextareaField } from "@/components/ui/textarea-field";
import { Alert } from "@/components/ui/alert";
import {
  STORY_THEMES,
  type BehavioralStory,
  type BehavioralStoryUpsert,
  type StoryTheme,
} from "@/lib/api/behavioral";

export interface StarStoryEditorProps {
  initial?: BehavioralStory;
  saving?: boolean;
  improving?: boolean;
  /** Validation/save error to surface inline. */
  error?: string | null;
  onSubmit: (payload: BehavioralStoryUpsert) => void;
  /** Only available when editing an existing story. */
  onImprove?: () => void;
  onCancel: () => void;
}

/**
 * The STAR story editor (DESIGN-SYSTEM §4.8): title + theme + tags and four
 * auto-grow Situation / Task / Action / Result sections, plus metrics and an
 * "Improve with AI" action (existing stories only).
 */
export function StarStoryEditor({
  initial,
  saving = false,
  improving = false,
  error,
  onSubmit,
  onImprove,
  onCancel,
}: StarStoryEditorProps) {
  const [title, setTitle] = React.useState(initial?.title ?? "");
  const [theme, setTheme] = React.useState<StoryTheme | "">(initial?.theme ?? "");
  const [situation, setSituation] = React.useState(initial?.situation ?? "");
  const [task, setTask] = React.useState(initial?.task ?? "");
  const [action, setAction] = React.useState(initial?.action ?? "");
  const [result, setResult] = React.useState(initial?.result ?? "");
  const [metrics, setMetrics] = React.useState(initial?.metrics ?? "");
  const [tagsText, setTagsText] = React.useState((initial?.tags ?? []).join(", "));
  const [titleError, setTitleError] = React.useState<string>();
  const [themeError, setThemeError] = React.useState<string>();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    let ok = true;
    if (!title.trim()) {
      setTitleError("Give your story a short title.");
      ok = false;
    } else setTitleError(undefined);
    if (!theme) {
      setThemeError("Pick the theme this story answers.");
      ok = false;
    } else setThemeError(undefined);
    if (!ok) return;

    const tags = tagsText
      .split(",")
      .map((t) => t.trim())
      .filter(Boolean);

    onSubmit({
      title: title.trim(),
      theme: theme as StoryTheme,
      situation: situation.trim() || undefined,
      task: task.trim() || undefined,
      action: action.trim() || undefined,
      result: result.trim() || undefined,
      metrics: metrics.trim() || undefined,
      tags,
    });
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && <Alert variant="danger">{error}</Alert>}

      <FormField
        id="story-title"
        label="Title"
        required
        placeholder='e.g. "Rescued the payments migration"'
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        error={titleError}
      />

      <SelectField
        id="story-theme"
        label="Theme"
        required
        value={theme}
        onChange={(e) => setTheme(e.target.value as StoryTheme)}
        error={themeError}
      >
        <option value="">Select a theme…</option>
        {STORY_THEMES.map((t) => (
          <option key={t.value} value={t.value}>
            {t.label}
          </option>
        ))}
      </SelectField>

      <TextareaField
        id="story-situation"
        label="Situation"
        rows={2}
        placeholder="Set the scene — context, constraints, stakes."
        value={situation}
        onChange={(e) => setSituation(e.target.value)}
      />
      <TextareaField
        id="story-task"
        label="Task"
        rows={2}
        placeholder="What were you specifically responsible for?"
        value={task}
        onChange={(e) => setTask(e.target.value)}
      />
      <TextareaField
        id="story-action"
        label="Action"
        rows={3}
        placeholder="What did you do? Lead with your decisions and trade-offs."
        value={action}
        onChange={(e) => setAction(e.target.value)}
      />
      <TextareaField
        id="story-result"
        label="Result"
        rows={2}
        placeholder="The outcome — quantified wherever possible."
        value={result}
        onChange={(e) => setResult(e.target.value)}
      />
      <FormField
        id="story-metrics"
        label="Metrics"
        placeholder='e.g. "cut p99 latency 40%, saved 12 eng-hours/wk"'
        hint="Numbers make a story land. Add the ones that matter."
        value={metrics}
        onChange={(e) => setMetrics(e.target.value)}
      />
      <FormField
        id="story-tags"
        label="Tags"
        placeholder="comma-separated, e.g. leadership, migration, on-call"
        value={tagsText}
        onChange={(e) => setTagsText(e.target.value)}
      />

      <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
        <div>
          {onImprove && (
            <Button
              type="button"
              variant="ghost"
              onClick={onImprove}
              loading={improving}
              disabled={saving}
            >
              <Sparkles aria-hidden /> Improve with AI
            </Button>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" onClick={onCancel} disabled={saving}>
            Cancel
          </Button>
          <Button type="submit" loading={saving} disabled={improving}>
            {initial ? "Save story" : "Create story"}
          </Button>
        </div>
      </div>
    </form>
  );
}
