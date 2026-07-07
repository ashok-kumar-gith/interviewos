"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { Building2, ExternalLink, Lightbulb, ListChecks, Shapes, Sparkles, Terminal, TriangleAlert } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { DifficultyPill } from "@/components/ui/difficulty-pill";
import {
  BackLink,
  DetailError,
  DetailSection,
  DetailSkeleton,
  MarkdownSection,
} from "@/components/detail/detail-layout";
import { getProblem, type ProblemDetail } from "@/lib/api/content";
import { ApiError } from "@/lib/api/client";
import { ProblemProgressPanel } from "@/components/detail/problem-progress";
import { CodeRunner } from "@/components/code/code-runner";
import { ProblemAdminActions } from "@/components/authoring/admin-actions";
import { AiReviewPanel } from "@/components/ai/ai-review-panel";
import { CODE_RUNNER_ENABLED } from "@/lib/config";

const SOURCE_LABEL: Record<string, string> = {
  blind75: "Blind 75",
  neetcode150: "NeetCode 150",
  grind75: "Grind 75",
  tech_interview_handbook: "Tech Interview Handbook",
  leetcode_top: "LeetCode Top",
  striver_sde: "Striver SDE Sheet",
  custom: "Custom",
};

export default function ProblemDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = React.use(params);
  const query = useQuery<ProblemDetail, unknown>({
    queryKey: ["problem", id],
    queryFn: () => getProblem(id),
    retry: (count, error) => !(error instanceof ApiError && error.status === 404) && count < 1,
  });

  if (query.isLoading) return <DetailSkeleton />;
  if (query.isError || !query.data) {
    return (
      <DetailError
        notFound={query.error instanceof ApiError && query.error.status === 404}
        backHref="/problems"
        backLabel="Back to Problems"
        onRetry={() => query.refetch()}
      />
    );
  }

  const p = query.data;
  const sortedFreq = [...(p.company_frequency ?? [])].sort((a, b) => b.frequency - a.frequency);

  return (
    <article className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <BackLink href="/problems" label="Back to Problems" />
        <ProblemAdminActions problem={p} />
      </div>

      <header className="space-y-3">
        <div className="flex flex-wrap items-center gap-2">
          <DifficultyPill difficulty={p.difficulty} />
          {p.is_premium && (
            <Badge variant="warning" size="sm">
              Premium
            </Badge>
          )}
          {p.platform && (
            <Badge variant="outline" size="sm" className="capitalize">
              {p.platform}
            </Badge>
          )}
        </div>
        <h1 className="text-h1">{p.title}</h1>
        <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
          {p.estimated_minutes != null && (
            <span className="tabular-nums">~{p.estimated_minutes} min</span>
          )}
          {p.frequency_score != null && (
            <span className="tabular-nums">Frequency {p.frequency_score.toFixed(1)}</span>
          )}
          {p.url && (
            <a
              href={p.url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 font-medium text-primary underline-offset-4 hover:underline"
            >
              Open on {p.platform ? p.platform[0].toUpperCase() + p.platform.slice(1) : "platform"}
              <ExternalLink className="size-3.5" aria-hidden />
            </a>
          )}
        </div>
      </header>

      <ProblemProgressPanel problemId={id} />

      {CODE_RUNNER_ENABLED && (
        <DetailSection title="Run code" icon={Terminal}>
          <CodeRunner />
        </DetailSection>
      )}

      <DetailSection title="Review with AI" icon={Sparkles}>
        <AiReviewPanel kind="code" problemTitle={p.title} prompt={p.prompt_summary} />
      </DetailSection>

      <MarkdownSection title="Problem" icon={ListChecks} content={p.prompt_summary} />
      <MarkdownSection title="Approach" icon={Lightbulb} content={p.approach_md} />
      <MarkdownSection title="Common mistakes" icon={TriangleAlert} content={p.common_mistakes} />

      {p.patterns && p.patterns.length > 0 && (
        <DetailSection title="Patterns" icon={Shapes}>
          <div className="space-y-3">
            {p.patterns.map((pat) => (
              <div key={pat.id} className="rounded-md border border-border p-3">
                <p className="text-sm font-medium">{pat.name}</p>
                {pat.when_to_use && (
                  <p className="mt-1 text-sm text-muted-foreground">{pat.when_to_use}</p>
                )}
                {!pat.when_to_use && pat.description && (
                  <p className="mt-1 text-sm text-muted-foreground">{pat.description}</p>
                )}
              </div>
            ))}
          </div>
        </DetailSection>
      )}

      {sortedFreq.length > 0 && (
        <DetailSection title="Company frequency" icon={Building2}>
          <ul className="space-y-2">
            {sortedFreq.map((c) => {
              const pct = Math.max(0, Math.min(100, Math.round(c.frequency * 100)));
              return (
                <li key={c.company_id} className="flex items-center gap-3">
                  <span className="w-40 shrink-0 truncate text-sm font-medium">
                    {c.company_name}
                  </span>
                  <span
                    className="h-2 flex-1 overflow-hidden rounded-full bg-muted"
                    role="img"
                    aria-label={`${c.company_name}: ${pct}% frequency`}
                  >
                    <span
                      className="block h-full rounded-full bg-primary"
                      style={{ width: `${pct}%` }}
                    />
                  </span>
                  <span className="w-10 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
                    {pct}%
                  </span>
                  {c.last_seen_period && (
                    <span className="hidden w-24 shrink-0 text-right text-xs text-muted-foreground sm:inline">
                      {c.last_seen_period}
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        </DetailSection>
      )}

      {p.sources && p.sources.length > 0 && (
        <DetailSection title="Sources">
          <ul className="flex flex-wrap gap-2">
            {p.sources.map((s, i) => {
              const label = SOURCE_LABEL[s.source] ?? s.source;
              const text = s.source_rank != null ? `${label} · #${s.source_rank}` : label;
              return (
                <li key={`${s.source}-${i}`}>
                  {s.source_url ? (
                    <a
                      href={s.source_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1.5 rounded-sm border border-border px-2.5 py-1 text-xs font-medium transition-colors hover:bg-muted"
                    >
                      {text}
                      <ExternalLink className="size-3" aria-hidden />
                    </a>
                  ) : (
                    <span className="inline-flex rounded-sm border border-border px-2.5 py-1 text-xs font-medium">
                      {text}
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        </DetailSection>
      )}
    </article>
  );
}
