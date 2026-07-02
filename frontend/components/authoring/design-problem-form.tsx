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
import type { DesignProblemDetail } from "@/lib/api/content";
import type { DesignProblemWrite } from "@/lib/api/designproblems";
import type { Difficulty } from "@/lib/api/types";

/** The HLD section fields, in the order they appear on the detail page. */
const SECTIONS: { key: keyof DesignProblemWrite; label: string }[] = [
  { key: "requirements_md", label: "Requirements" },
  { key: "capacity_estimation_md", label: "Capacity estimation" },
  { key: "api_design_md", label: "API design" },
  { key: "data_model_md", label: "Data model" },
  { key: "high_level_design_md", label: "High-level design" },
  { key: "caching_md", label: "Caching" },
  { key: "queueing_md", label: "Queueing" },
  { key: "scaling_md", label: "Scaling" },
  { key: "tradeoffs_md", label: "Tradeoffs" },
  { key: "failure_handling_md", label: "Failure handling" },
  { key: "alternatives_md", label: "Alternatives" },
  { key: "interview_tips_md", label: "Interview tips" },
];

type SectionKey = (typeof SECTIONS)[number]["key"];

interface DesignFormValues {
  slug: string;
  title: string;
  difficulty: Difficulty;
  order_index: string;
  requirements_md: string;
  capacity_estimation_md: string;
  api_design_md: string;
  data_model_md: string;
  high_level_design_md: string;
  caching_md: string;
  queueing_md: string;
  scaling_md: string;
  tradeoffs_md: string;
  failure_handling_md: string;
  alternatives_md: string;
  interview_tips_md: string;
}

export interface DesignProblemFormProps {
  initial?: DesignProblemDetail;
  onSubmit: (body: DesignProblemWrite) => void;
  submitting: boolean;
  serverError?: string | null;
  fieldErrors?: Record<string, string>;
  submitLabel: string;
  onCancel?: () => void;
}

export function DesignProblemForm({
  initial,
  onSubmit,
  submitting,
  serverError,
  fieldErrors = {},
  submitLabel,
  onCancel,
}: DesignProblemFormProps) {
  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<DesignFormValues>({
    defaultValues: {
      slug: initial?.slug ?? "",
      title: initial?.title ?? "",
      difficulty: initial?.difficulty ?? "medium",
      order_index: initial?.order_index != null ? String(initial.order_index) : "",
      requirements_md: initial?.requirements_md ?? "",
      capacity_estimation_md: initial?.capacity_estimation_md ?? "",
      api_design_md: initial?.api_design_md ?? "",
      data_model_md: initial?.data_model_md ?? "",
      high_level_design_md: initial?.high_level_design_md ?? "",
      caching_md: initial?.caching_md ?? "",
      queueing_md: initial?.queueing_md ?? "",
      scaling_md: initial?.scaling_md ?? "",
      tradeoffs_md: initial?.tradeoffs_md ?? "",
      failure_handling_md: initial?.failure_handling_md ?? "",
      alternatives_md: initial?.alternatives_md ?? "",
      interview_tips_md: initial?.interview_tips_md ?? "",
    },
  });

  const [followUps, setFollowUps] = React.useState<string[]>(initial?.follow_up_questions ?? []);
  const slugTouched = React.useRef(Boolean(initial));

  const submit = handleSubmit((values) => {
    const body: DesignProblemWrite = {
      slug: values.slug.trim(),
      title: values.title.trim(),
      difficulty: values.difficulty,
      order_index: parseOptionalNumber(values.order_index),
      follow_up_questions: followUps,
    };
    for (const { key } of SECTIONS) {
      body[key] = orNull(values[key as keyof DesignFormValues] as string) as never;
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
          placeholder="Design a URL shortener"
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
          placeholder="design-url-shortener"
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

      {SECTIONS.map(({ key, label }) => (
        <MarkdownTextarea
          key={key}
          id={key as string}
          label={label}
          error={fieldErrors[key as string]}
          {...register(key as SectionKey as keyof DesignFormValues)}
        />
      ))}

      <TagInput
        id="follow_up_questions"
        label="Follow-up questions"
        value={followUps}
        onChange={setFollowUps}
        placeholder="How would you handle custom aliases?"
        hint="Press Enter after each question."
      />
    </FormShell>
  );
}
