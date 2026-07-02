"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { BookOpen, HelpCircle, Library, ListTree, TriangleAlert } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { DifficultyPill } from "@/components/ui/difficulty-pill";
import { Markdown } from "@/components/ui/markdown";
import { ResourceRow } from "@/components/catalog/resource-row";
import {
  BackLink,
  DetailError,
  DetailSection,
  DetailSkeleton,
  MarkdownSection,
} from "@/components/detail/detail-layout";
import { getBackendEngineeringTopic, type TopicDetail } from "@/lib/api/content";
import { ApiError } from "@/lib/api/client";
import { TopicAdminActions } from "@/components/authoring/admin-actions";

const STATUS_LABEL: Record<string, string> = {
  not_started: "Not started",
  in_progress: "In progress",
  completed: "Completed",
  needs_review: "Needs review",
};

export default function BackendEngineeringTopicDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = React.use(params);
  const query = useQuery<TopicDetail, unknown>({
    queryKey: ["backend-engineering-topic", id],
    queryFn: () => getBackendEngineeringTopic(id),
    retry: (count, error) => !(error instanceof ApiError && error.status === 404) && count < 1,
  });

  if (query.isLoading) return <DetailSkeleton />;
  if (query.isError || !query.data) {
    return (
      <DetailError
        notFound={query.error instanceof ApiError && query.error.status === 404}
        backHref="/backend-engineering"
        backLabel="Back to Backend Engineering"
        onRetry={() => query.refetch()}
      />
    );
  }

  const t = query.data;

  return (
    <article className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <BackLink href="/backend-engineering" label="Back to Backend Engineering" />
        <TopicAdminActions
          topic={t}
          queryKey={["backend-engineering-topic", id]}
          backHref="/backend-engineering"
        />
      </div>

      <header className="space-y-3">
        <div className="flex flex-wrap items-center gap-2">
          <DifficultyPill difficulty={t.difficulty} />
          <Badge variant="outline" size="sm" className="capitalize">
            {t.priority} priority
          </Badge>
          {t.estimated_hours != null && (
            <Badge variant="secondary" size="sm">
              ~{t.estimated_hours}h
            </Badge>
          )}
          {t.progress?.status && (
            <Badge
              variant={t.progress.status === "completed" ? "success" : "secondary"}
              size="sm"
            >
              {STATUS_LABEL[t.progress.status] ?? t.progress.status}
            </Badge>
          )}
        </div>
        <h1 className="text-h1">{t.name}</h1>
        {t.summary && <p className="text-sm text-muted-foreground">{t.summary}</p>}
      </header>

      <MarkdownSection title="Concept" icon={BookOpen} content={t.concept_md} />
      <MarkdownSection title="Common mistakes" icon={TriangleAlert} content={t.common_mistakes} />

      {t.subtopics && t.subtopics.length > 0 && (
        <DetailSection title="Subtopics" icon={ListTree}>
          <div className="space-y-4">
            {t.subtopics.map((s) => (
              <div key={s.id} className="rounded-md border border-border p-3">
                <p className="text-sm font-medium">{s.name}</p>
                {s.content_md && <Markdown className="mt-2" content={s.content_md} />}
              </div>
            ))}
          </div>
        </DetailSection>
      )}

      {t.expected_questions && t.expected_questions.length > 0 && (
        <DetailSection title="Expected questions" icon={HelpCircle}>
          <ul className="list-disc space-y-1.5 pl-5 text-sm marker:text-muted-foreground">
            {t.expected_questions.map((q, i) => (
              <li key={i}>{q}</li>
            ))}
          </ul>
        </DetailSection>
      )}

      {t.resources && t.resources.length > 0 && (
        <DetailSection title="Resources" icon={Library}>
          <div className="space-y-2">
            {t.resources.map((r) => (
              <ResourceRow key={r.id} resource={r} />
            ))}
          </div>
        </DetailSection>
      )}
    </article>
  );
}
