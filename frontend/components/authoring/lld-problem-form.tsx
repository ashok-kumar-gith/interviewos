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
  orNull,
  parseOptionalNumber,
  slugify,
  slugSchema,
  titleSchema,
} from "@/components/authoring/shared";
import { zodValidate } from "@/lib/form/zod-rules";
import type { LLDProblemDetail } from "@/lib/api/content";
import type { LLDProblemWrite } from "@/lib/api/lld";
import type { Difficulty } from "@/lib/api/types";

const SECTIONS: { key: keyof LLDProblemWrite; label: string }[] = [
  { key: "requirements_md", label: "Requirements" },
  { key: "entities_md", label: "Entities" },
  { key: "class_diagram_md", label: "Class diagram (UML / mermaid)" },
  { key: "solid_notes_md", label: "SOLID notes" },
  { key: "api_or_interface_md", label: "API / interface" },
  { key: "tradeoffs_md", label: "Tradeoffs" },
];

interface LLDFormValues {
  slug: string;
  title: string;
  difficulty: Difficulty;
  order_index: string;
  requirements_md: string;
  entities_md: string;
  class_diagram_md: string;
  solid_notes_md: string;
  api_or_interface_md: string;
  tradeoffs_md: string;
}

export interface LLDProblemFormProps {
  initial?: LLDProblemDetail;
  onSubmit: (body: LLDProblemWrite) => void;
  submitting: boolean;
  serverError?: string | null;
  fieldErrors?: Record<string, string>;
  submitLabel: string;
  onCancel?: () => void;
}

export function LLDProblemForm({
  initial,
  onSubmit,
  submitting,
  serverError,
  fieldErrors = {},
  submitLabel,
  onCancel,
}: LLDProblemFormProps) {
  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<LLDFormValues>({
    defaultValues: {
      slug: initial?.slug ?? "",
      title: initial?.title ?? "",
      difficulty: initial?.difficulty ?? "medium",
      order_index: initial?.order_index != null ? String(initial.order_index) : "",
      requirements_md: initial?.requirements_md ?? "",
      entities_md: initial?.entities_md ?? "",
      class_diagram_md: initial?.class_diagram_md ?? "",
      solid_notes_md: initial?.solid_notes_md ?? "",
      api_or_interface_md: initial?.api_or_interface_md ?? "",
      tradeoffs_md: initial?.tradeoffs_md ?? "",
    },
  });

  const [designPatterns, setDesignPatterns] = React.useState<string[]>(
    initial?.design_patterns ?? [],
  );
  const [followUps, setFollowUps] = React.useState<string[]>(initial?.follow_up_questions ?? []);
  const slugTouched = React.useRef(Boolean(initial));

  const submit = handleSubmit((values) => {
    const body: LLDProblemWrite = {
      slug: values.slug.trim(),
      title: values.title.trim(),
      difficulty: values.difficulty,
      order_index: parseOptionalNumber(values.order_index),
      design_patterns: designPatterns,
      follow_up_questions: followUps,
    };
    for (const { key } of SECTIONS) {
      body[key] = orNull(values[key as keyof LLDFormValues] as string) as never;
    }
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
        <FormField
          id="title"
          label="Title"
          required
          placeholder="Design a parking lot"
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
          placeholder="design-parking-lot"
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
        <FormField
          id="order_index"
          label="Order index"
          type="number"
          min={0}
          hint="Position in the ordered catalog."
          error={fieldErrors.order_index}
          {...register("order_index")}
        />
      </div>

      <TagInput
        id="design_patterns"
        label="Design patterns"
        value={designPatterns}
        onChange={setDesignPatterns}
        placeholder="Factory, Strategy, Observer…"
      />

      {SECTIONS.map(({ key, label }) => (
        <MarkdownTextarea
          key={key}
          id={key as string}
          label={label}
          error={fieldErrors[key as string]}
          {...register(key as keyof LLDFormValues)}
        />
      ))}

      <TagInput
        id="follow_up_questions"
        label="Follow-up questions"
        value={followUps}
        onChange={setFollowUps}
        placeholder="How would you support electric-vehicle charging spots?"
        hint="Press Enter after each question."
      />
    </FormShell>
  );
}
