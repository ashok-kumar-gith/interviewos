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
  PILLAR_OPTIONS,
  PRIORITIES,
  nameSchema,
  orNull,
  parseOptionalNumber,
  slugify,
  slugSchema,
} from "@/components/authoring/shared";
import { zodValidate } from "@/lib/form/zod-rules";
import type { TopicDetail, TopicWrite } from "@/lib/api/content";
import type { Difficulty, PillarType, Priority } from "@/lib/api/types";

interface TopicFormValues {
  pillar_type: PillarType | "";
  slug: string;
  name: string;
  summary: string;
  difficulty: Difficulty;
  priority: Priority;
  estimated_hours: string;
  sort_order: string;
  concept_md: string;
  common_mistakes: string;
}

export interface TopicFormProps {
  initial?: TopicDetail;
  /** On edit the pillar is already fixed; hide the picker. */
  editing?: boolean;
  onSubmit: (body: TopicWrite) => void;
  submitting: boolean;
  serverError?: string | null;
  fieldErrors?: Record<string, string>;
  submitLabel: string;
  onCancel?: () => void;
}

export function TopicForm({
  initial,
  editing = false,
  onSubmit,
  submitting,
  serverError,
  fieldErrors = {},
  submitLabel,
  onCancel,
}: TopicFormProps) {
  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<TopicFormValues>({
    defaultValues: {
      pillar_type: "",
      slug: initial?.slug ?? "",
      name: initial?.name ?? "",
      summary: initial?.summary ?? "",
      difficulty: initial?.difficulty ?? "medium",
      priority: initial?.priority ?? "medium",
      estimated_hours: initial?.estimated_hours != null ? String(initial.estimated_hours) : "",
      sort_order: initial?.sort_order != null ? String(initial.sort_order) : "",
      concept_md: initial?.concept_md ?? "",
      common_mistakes: initial?.common_mistakes ?? "",
    },
  });

  const [expectedQuestions, setExpectedQuestions] = React.useState<string[]>(
    initial?.expected_questions ?? [],
  );
  const [prerequisites, setPrerequisites] = React.useState<string[]>(initial?.prerequisites ?? []);
  const slugTouched = React.useRef(Boolean(initial));

  const submit = handleSubmit((values) => {
    const body: TopicWrite = {
      slug: values.slug.trim(),
      name: values.name.trim(),
      summary: orNull(values.summary),
      difficulty: values.difficulty,
      priority: values.priority,
      estimated_hours: parseOptionalNumber(values.estimated_hours),
      sort_order: parseOptionalNumber(values.sort_order),
      concept_md: orNull(values.concept_md),
      common_mistakes: orNull(values.common_mistakes),
      expected_questions: expectedQuestions,
      prerequisites,
    };
    // On create the pillar is required (pillar_type). On edit the topic keeps
    // its existing pillar, so we omit it (the picker isn't shown).
    if (!editing && values.pillar_type) body.pillar_type = values.pillar_type;
    onSubmit(body);
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
        {!editing && (
          <SelectField
            id="pillar_type"
            label="Pillar"
            required
            containerClassName="sm:col-span-2"
            error={fieldErrors.pillar_type ?? fieldErrors.pillar_id}
            {...register("pillar_type", {
              validate: (v) => (v ? true : "Choose a pillar"),
            })}
          >
            <option value="">— Choose a pillar —</option>
            {PILLAR_OPTIONS.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </SelectField>
        )}
        <FormField
          id="name"
          label="Name"
          required
          placeholder="Consistent hashing"
          error={errors.name?.message ?? fieldErrors.name}
          {...register("name", {
            validate: zodValidate(nameSchema),
            onChange: (e) => {
              if (!slugTouched.current) setValue("slug", slugify(e.target.value));
            },
          })}
        />
        <FormField
          id="slug"
          label="Slug"
          required
          placeholder="consistent-hashing"
          error={errors.slug?.message ?? fieldErrors.slug}
          {...register("slug", {
            validate: zodValidate(slugSchema),
            onChange: () => {
              slugTouched.current = true;
            },
          })}
        />
        <SelectField id="difficulty" label="Difficulty" {...register("difficulty")}>
          {DIFFICULTIES.map((d) => (
            <option key={d} value={d}>
              {d[0].toUpperCase() + d.slice(1)}
            </option>
          ))}
        </SelectField>
        <SelectField id="priority" label="Priority" {...register("priority")}>
          {PRIORITIES.map((p) => (
            <option key={p} value={p}>
              {p[0].toUpperCase() + p.slice(1)}
            </option>
          ))}
        </SelectField>
        <FormField
          id="estimated_hours"
          label="Estimated hours"
          type="number"
          step="0.5"
          min={0}
          error={fieldErrors.estimated_hours}
          {...register("estimated_hours")}
        />
        <FormField
          id="sort_order"
          label="Sort order"
          type="number"
          min={0}
          error={fieldErrors.sort_order}
          {...register("sort_order")}
        />
      </div>

      <FormField
        id="summary"
        label="Summary"
        placeholder="One-line description shown in the catalog."
        error={fieldErrors.summary}
        {...register("summary")}
      />

      <MarkdownTextarea
        id="concept_md"
        label="Concept"
        error={fieldErrors.concept_md}
        {...register("concept_md")}
      />
      <MarkdownTextarea
        id="common_mistakes"
        label="Common mistakes"
        error={fieldErrors.common_mistakes}
        {...register("common_mistakes")}
      />

      <TagInput
        id="prerequisites"
        label="Prerequisites"
        value={prerequisites}
        onChange={setPrerequisites}
        placeholder="Slugs or names of prerequisite topics."
      />
      <TagInput
        id="expected_questions"
        label="Expected questions"
        value={expectedQuestions}
        onChange={setExpectedQuestions}
        placeholder="What interview questions test this topic?"
        hint="Press Enter after each question."
      />
    </FormShell>
  );
}
