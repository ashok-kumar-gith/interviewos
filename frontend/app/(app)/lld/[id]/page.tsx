"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Boxes,
  GitFork,
  HelpCircle,
  ListChecks,
  Network,
  Shapes,
  SquareStack,
  Webhook,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { DifficultyPill } from "@/components/ui/difficulty-pill";
import {
  BackLink,
  DetailError,
  DetailSection,
  DetailSkeleton,
  MarkdownSection,
} from "@/components/detail/detail-layout";
import { getLLDProblem, type LLDProblemDetail } from "@/lib/api/content";
import { ApiError } from "@/lib/api/client";

/** Strip a fenced ``` wrapper if the diagram is already fenced. */
function unfence(md: string): string {
  const m = md.match(/^```[^\n]*\n([\s\S]*?)\n```\s*$/);
  return m ? m[1] : md;
}

export default function LLDProblemDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = React.use(params);
  const query = useQuery<LLDProblemDetail, unknown>({
    queryKey: ["lld-problem", id],
    queryFn: () => getLLDProblem(id),
    retry: (count, error) => !(error instanceof ApiError && error.status === 404) && count < 1,
  });

  if (query.isLoading) return <DetailSkeleton />;
  if (query.isError || !query.data) {
    return (
      <DetailError
        notFound={query.error instanceof ApiError && query.error.status === 404}
        backHref="/lld"
        backLabel="Back to LLD"
        onRetry={() => query.refetch()}
      />
    );
  }

  const l = query.data;

  return (
    <article className="space-y-6">
      <BackLink href="/lld" label="Back to LLD" />

      <header className="space-y-3">
        <DifficultyPill difficulty={l.difficulty} />
        <h1 className="text-h1">{l.title}</h1>
      </header>

      <MarkdownSection title="Requirements" icon={ListChecks} content={l.requirements_md} />
      <MarkdownSection title="Entities" icon={Boxes} content={l.entities_md} />

      {l.design_patterns && l.design_patterns.length > 0 && (
        <DetailSection title="Design patterns" icon={Shapes}>
          <ul className="flex flex-wrap gap-2">
            {l.design_patterns.map((p, i) => (
              <li key={i}>
                <Badge variant="secondary">{p}</Badge>
              </li>
            ))}
          </ul>
        </DetailSection>
      )}

      {l.class_diagram_md && l.class_diagram_md.trim() !== "" && (
        <DetailSection title="Class diagram" icon={Network}>
          <p className="mb-2 text-xs text-muted-foreground">
            UML / mermaid source — paste into a mermaid renderer to visualize.
          </p>
          <pre className="overflow-x-auto rounded-md border border-border bg-muted/50 p-3">
            <code className="font-mono text-xs leading-relaxed text-foreground">
              {unfence(l.class_diagram_md)}
            </code>
          </pre>
        </DetailSection>
      )}

      <MarkdownSection title="SOLID notes" icon={SquareStack} content={l.solid_notes_md} />
      <MarkdownSection title="API / interface" icon={Webhook} content={l.api_or_interface_md} />
      <MarkdownSection title="Tradeoffs" icon={GitFork} content={l.tradeoffs_md} />

      {l.follow_up_questions && l.follow_up_questions.length > 0 && (
        <DetailSection title="Follow-up questions" icon={HelpCircle}>
          <ul className="list-disc space-y-1.5 pl-5 text-sm marker:text-muted-foreground">
            {l.follow_up_questions.map((q, i) => (
              <li key={i}>{q}</li>
            ))}
          </ul>
        </DetailSection>
      )}
    </article>
  );
}
