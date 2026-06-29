"use client";

import * as React from "react";

import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { TextareaField } from "@/components/ui/textarea-field";
import { Alert } from "@/components/ui/alert";
import type { ResumeProject, ResumeProjectUpsert } from "@/lib/api/resume";

export interface ResumeProjectEditorProps {
  initial?: ResumeProject;
  saving?: boolean;
  error?: string | null;
  onSubmit: (payload: ResumeProjectUpsert) => void;
  onCancel: () => void;
}

/** Editor for a single resume project — impact bullets + tech stack. */
export function ResumeProjectEditor({
  initial,
  saving = false,
  error,
  onSubmit,
  onCancel,
}: ResumeProjectEditorProps) {
  const [name, setName] = React.useState(initial?.name ?? "");
  const [role, setRole] = React.useState(initial?.role ?? "");
  const [description, setDescription] = React.useState(initial?.description ?? "");
  const [impact, setImpact] = React.useState(initial?.impact ?? "");
  const [metricsText, setMetricsText] = React.useState((initial?.metrics ?? []).join("\n"));
  const [techText, setTechText] = React.useState((initial?.tech_stack ?? []).join(", "));
  const [nameError, setNameError] = React.useState<string>();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      setNameError("Name your project.");
      return;
    }
    setNameError(undefined);

    onSubmit({
      name: name.trim(),
      role: role.trim() || undefined,
      description: description.trim() || undefined,
      impact: impact.trim() || undefined,
      metrics: metricsText
        .split("\n")
        .map((m) => m.trim())
        .filter(Boolean),
      tech_stack: techText
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean),
    });
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && <Alert variant="danger">{error}</Alert>}
      <FormField
        id="project-name"
        label="Project / role title"
        required
        placeholder='e.g. "Realtime fraud detection pipeline"'
        value={name}
        onChange={(e) => setName(e.target.value)}
        error={nameError}
      />
      <FormField
        id="project-role"
        label="Your role"
        placeholder='e.g. "Tech lead"'
        value={role}
        onChange={(e) => setRole(e.target.value)}
      />
      <TextareaField
        id="project-description"
        label="Description"
        rows={2}
        placeholder="What the project was and why it mattered."
        value={description}
        onChange={(e) => setDescription(e.target.value)}
      />
      <TextareaField
        id="project-impact"
        label="Impact bullets"
        rows={3}
        placeholder="Lead with the outcome and your contribution."
        value={impact}
        onChange={(e) => setImpact(e.target.value)}
      />
      <TextareaField
        id="project-metrics"
        label="Metrics"
        rows={2}
        hint="One per line — e.g. cut latency 40%, $1.2M saved."
        value={metricsText}
        onChange={(e) => setMetricsText(e.target.value)}
      />
      <FormField
        id="project-tech"
        label="Tech stack"
        placeholder="comma-separated, e.g. Go, Kafka, Postgres"
        value={techText}
        onChange={(e) => setTechText(e.target.value)}
      />
      <div className="flex items-center justify-end gap-2 pt-1">
        <Button type="button" variant="outline" onClick={onCancel} disabled={saving}>
          Cancel
        </Button>
        <Button type="submit" loading={saving}>
          {initial ? "Save project" : "Add project"}
        </Button>
      </div>
    </form>
  );
}
