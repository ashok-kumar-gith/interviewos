"use client";

import * as React from "react";
import { useForm } from "react-hook-form";

import { FormField } from "@/components/ui/form-field";
import { SelectField } from "@/components/ui/select-field";
import { MarkdownTextarea } from "@/components/ui/markdown-textarea";
import { TagInput } from "@/components/ui/tag-input";
import { FormShell } from "@/components/authoring/form-shell";
import {
  DIFFICULTIES,
  PROBLEM_PLATFORMS,
  PROBLEM_SOURCES,
  orNull,
  parseOptionalNumber,
  slugify,
  slugSchema,
  titleSchema,
} from "@/components/authoring/shared";
import { zodValidate } from "@/lib/form/zod-rules";
import type { ProblemDetail, ProblemWrite } from "@/lib/api/content";
import type { Difficulty, ProblemPlatform, ProblemSourceName } from "@/lib/api/types";

interface ProblemFormValues {
  slug: string;
  title: string;
  difficulty: Difficulty;
  platform: string;
  url: string;
  external_id: string;
  estimated_minutes: string;
  frequency_score: string;
  is_premium: boolean;
  prompt_summary: string;
  approach_md: string;
  common_mistakes: string;
}

export interface ProblemFormProps {
  initial?: ProblemDetail;
  onSubmit: (body: ProblemWrite) => void;
  submitting: boolean;
  serverError?: string | null;
  fieldErrors?: Record<string, string>;
  submitLabel: string;
  onCancel?: () => void;
}

export function ProblemForm({
  initial,
  onSubmit,
  submitting,
  serverError,
  fieldErrors = {},
  submitLabel,
  onCancel,
}: ProblemFormProps) {
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<ProblemFormValues>({
    defaultValues: {
      slug: initial?.slug ?? "",
      title: initial?.title ?? "",
      difficulty: initial?.difficulty ?? "medium",
      platform: initial?.platform ?? "",
      url: initial?.url ?? "",
      external_id: initial?.external_id ?? "",
      estimated_minutes: initial?.estimated_minutes != null ? String(initial.estimated_minutes) : "",
      frequency_score: initial?.frequency_score != null ? String(initial.frequency_score) : "",
      is_premium: initial?.is_premium ?? false,
      prompt_summary: initial?.prompt_summary ?? "",
      approach_md: initial?.approach_md ?? "",
      common_mistakes: initial?.common_mistakes ?? "",
    },
  });

  const [patternSlugs, setPatternSlugs] = React.useState<string[]>(
    initial?.patterns?.map((p) => p.slug) ?? [],
  );
  const [sources, setSources] = React.useState<ProblemSourceName[]>(
    initial?.sources?.map((s) => s.source) ?? [],
  );

  const slugTouched = React.useRef(Boolean(initial));

  const submit = handleSubmit((values) => {
    onSubmit({
      slug: values.slug.trim(),
      title: values.title.trim(),
      difficulty: values.difficulty,
      platform: values.platform ? (values.platform as ProblemPlatform) : undefined,
      url: orNull(values.url),
      external_id: orNull(values.external_id),
      estimated_minutes: parseOptionalNumber(values.estimated_minutes),
      frequency_score: parseOptionalNumber(values.frequency_score),
      is_premium: values.is_premium,
      prompt_summary: orNull(values.prompt_summary),
      approach_md: orNull(values.approach_md),
      common_mistakes: orNull(values.common_mistakes),
      pattern_slugs: patternSlugs,
      sources,
    });
  });

  return (
    <FormShell
      onSubmit={submit}
      submitting={submitting}
      submitLabel={submitLabel}
      error={serverError}
      onCancel={onCancel}
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <FormField
          id="title"
          label="Title"
          required
          placeholder="Two Sum"
          error={errors.title?.message ?? fieldErrors.title}
          {...register("title", {
            validate: zodValidate(titleSchema),
            onChange: (e) => {
              if (!slugTouched.current) setValue("slug", slugify(e.target.value));
            },
          })}
        />
        <FormField
          id="slug"
          label="Slug"
          required
          placeholder="two-sum"
          error={errors.slug?.message ?? fieldErrors.slug}
          {...register("slug", {
            validate: zodValidate(slugSchema),
            onChange: () => {
              slugTouched.current = true;
            },
          })}
        />
        <SelectField id="difficulty" label="Difficulty" required {...register("difficulty")}>
          {DIFFICULTIES.map((d) => (
            <option key={d} value={d}>
              {d[0].toUpperCase() + d.slice(1)}
            </option>
          ))}
        </SelectField>
        <SelectField id="platform" label="Platform" {...register("platform")}>
          <option value="">— None —</option>
          {PROBLEM_PLATFORMS.map((p) => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
        </SelectField>
        <FormField
          id="url"
          label="URL"
          type="url"
          placeholder="https://leetcode.com/problems/two-sum"
          error={fieldErrors.url}
          {...register("url")}
        />
        <FormField
          id="external_id"
          label="External ID"
          placeholder="1"
          error={fieldErrors.external_id}
          {...register("external_id")}
        />
        <FormField
          id="estimated_minutes"
          label="Estimated minutes"
          type="number"
          min={0}
          error={fieldErrors.estimated_minutes}
          {...register("estimated_minutes")}
        />
        <FormField
          id="frequency_score"
          label="Frequency score"
          type="number"
          step="0.1"
          min={0}
          error={fieldErrors.frequency_score}
          {...register("frequency_score")}
        />
      </div>

      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          className="size-4 rounded-sm border-input accent-primary"
          {...register("is_premium")}
        />
        Premium problem
      </label>

      <TagInput
        id="pattern_slugs"
        label="Pattern slugs"
        value={patternSlugs}
        onChange={setPatternSlugs}
        slugify
        placeholder="two-pointers, hash-map…"
        hint="Slugs of DSA patterns this problem exercises."
      />
      <TagInput
        id="sources"
        label="Sources"
        value={sources}
        onChange={(v) => setSources(v as ProblemSourceName[])}
        placeholder={PROBLEM_SOURCES.join(", ")}
        hint={`Known sources: ${PROBLEM_SOURCES.join(", ")}.`}
      />

      <MarkdownTextarea
        id="prompt_summary"
        label="Prompt summary"
        error={fieldErrors.prompt_summary}
        {...register("prompt_summary")}
      />
      <MarkdownTextarea
        id="approach_md"
        label="Approach"
        error={fieldErrors.approach_md}
        {...register("approach_md")}
      />
      <MarkdownTextarea
        id="common_mistakes"
        label="Common mistakes"
        error={fieldErrors.common_mistakes}
        {...register("common_mistakes")}
      />
    </FormShell>
  );
}
