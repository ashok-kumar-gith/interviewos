"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Boxes,
  Database,
  GitFork,
  HelpCircle,
  Layers,
  Lightbulb,
  ListChecks,
  Network,
  Repeat,
  Scale,
  ShieldAlert,
  TrendingUp,
  Webhook,
} from "lucide-react";

import { DifficultyPill } from "@/components/ui/difficulty-pill";
import {
  BackLink,
  DetailError,
  DetailSection,
  DetailSkeleton,
  MarkdownSection,
} from "@/components/detail/detail-layout";
import { getDesignProblem, type DesignProblemDetail } from "@/lib/api/content";
import { ApiError } from "@/lib/api/client";
import { DesignProblemProgressCard } from "@/components/detail/design-problem-progress";

export default function DesignProblemDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = React.use(params);
  const query = useQuery<DesignProblemDetail, unknown>({
    queryKey: ["design-problem", id],
    queryFn: () => getDesignProblem(id),
    retry: (count, error) => !(error instanceof ApiError && error.status === 404) && count < 1,
  });

  if (query.isLoading) return <DetailSkeleton />;
  if (query.isError || !query.data) {
    return (
      <DetailError
        notFound={query.error instanceof ApiError && query.error.status === 404}
        backHref="/system-design"
        backLabel="Back to System Design"
        onRetry={() => query.refetch()}
      />
    );
  }

  const d = query.data;

  return (
    <article className="space-y-6">
      <BackLink href="/system-design" label="Back to System Design" />

      <header className="space-y-3">
        <DifficultyPill difficulty={d.difficulty} />
        <h1 className="text-h1">{d.title}</h1>
      </header>

      <DesignProblemProgressCard id={id} />

      <MarkdownSection title="Requirements" icon={ListChecks} content={d.requirements_md} />
      <MarkdownSection
        title="Capacity estimation"
        icon={Scale}
        content={d.capacity_estimation_md}
      />
      <MarkdownSection title="API design" icon={Webhook} content={d.api_design_md} />
      <MarkdownSection title="Data model" icon={Database} content={d.data_model_md} />
      <MarkdownSection title="High-level design" icon={Network} content={d.high_level_design_md} />
      <MarkdownSection title="Caching" icon={Layers} content={d.caching_md} />
      <MarkdownSection title="Queueing" icon={Repeat} content={d.queueing_md} />
      <MarkdownSection title="Scaling" icon={TrendingUp} content={d.scaling_md} />
      <MarkdownSection title="Tradeoffs" icon={GitFork} content={d.tradeoffs_md} />
      <MarkdownSection
        title="Failure handling"
        icon={ShieldAlert}
        content={d.failure_handling_md}
      />
      <MarkdownSection title="Alternatives" icon={Boxes} content={d.alternatives_md} />
      <MarkdownSection title="Interview tips" icon={Lightbulb} content={d.interview_tips_md} />

      {d.follow_up_questions && d.follow_up_questions.length > 0 && (
        <DetailSection title="Follow-up questions" icon={HelpCircle}>
          <ul className="list-disc space-y-1.5 pl-5 text-sm marker:text-muted-foreground">
            {d.follow_up_questions.map((q, i) => (
              <li key={i}>{q}</li>
            ))}
          </ul>
        </DetailSection>
      )}
    </article>
  );
}
