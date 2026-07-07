"use client";

import * as React from "react";
import { useMutation } from "@tanstack/react-query";
import { RefreshCw, Sparkles, Wand2 } from "lucide-react";

import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Markdown } from "@/components/ui/markdown";
import {
  aiCodeReview,
  aiLldReview,
  aiSdReview,
  type AIResponse,
} from "@/lib/api/ai";
import { cn } from "@/lib/utils";

const CODE_LANGUAGES = [
  "python",
  "javascript",
  "typescript",
  "go",
  "java",
  "cpp",
  "c",
] as const;

export type ReviewKind = "code" | "lld" | "sd";

export interface AiReviewPanelProps {
  kind: ReviewKind;
  /** Problem title, passed to the model for context (code/lld). */
  problemTitle?: string | null;
  /** Problem prompt/requirements, passed for context (code/lld). */
  prompt?: string | null;
  /** Design-problem id — required for kind="sd". */
  designProblemId?: string;
  className?: string;
}

const COPY: Record<ReviewKind, { heading: string; blurb: string; placeholder: string; cta: string }> = {
  code: {
    heading: "AI code review",
    blurb: "Paste your solution and get a critique of correctness, complexity, edge cases, and style.",
    placeholder: "Paste your solution here…",
    cta: "Review my code",
  },
  lld: {
    heading: "AI low-level design review",
    blurb: "Describe your classes, responsibilities, and relationships to get an OOD critique.",
    placeholder:
      "Describe your design (Markdown): classes & attributes, responsibilities, relationships, patterns, edge cases…",
    cta: "Review my design",
  },
  sd: {
    heading: "AI system-design review",
    blurb: "Paste your design write-up to get a structured critique against a staff-level rubric.",
    placeholder:
      "Paste your system-design write-up (Markdown): requirements, capacity, API, data model, scaling, trade-offs…",
    cta: "Review my design",
  },
};

/**
 * AiReviewPanel — a self-contained "Review with AI" widget. Collects the user's
 * solution (code + language, or Markdown design) and renders the AI review. Works
 * with real Claude when enabled server-side, otherwise a deterministic rubric.
 */
export function AiReviewPanel({
  kind,
  problemTitle,
  prompt,
  designProblemId,
  className,
}: AiReviewPanelProps) {
  const copy = COPY[kind];
  const [text, setText] = React.useState("");
  const [language, setLanguage] = React.useState<string>("python");

  const mutation = useMutation<AIResponse, unknown, void>({
    mutationFn: () => {
      if (kind === "code") {
        return aiCodeReview({
          code: text,
          language,
          problem_title: problemTitle ?? undefined,
          prompt: prompt ?? undefined,
        });
      }
      if (kind === "lld") {
        return aiLldReview({
          answer_md: text,
          problem_title: problemTitle ?? undefined,
          prompt: prompt ?? undefined,
        });
      }
      return aiSdReview({ design_problem_id: designProblemId ?? "", answer_md: text });
    },
  });

  const canSubmit = text.trim().length > 0 && (kind !== "sd" || Boolean(designProblemId));

  return (
    <div className={cn("space-y-4", className)}>
      <Card>
        <CardContent className="space-y-3 py-5">
          <div>
            <h3 className="text-h3 font-semibold">{copy.heading}</h3>
            <p className="mt-0.5 text-sm text-muted-foreground">{copy.blurb}</p>
          </div>

          {kind === "code" && (
            <div className="space-y-1.5">
              <label htmlFor="review-language" className="text-sm font-medium">
                Language
              </label>
              <Select
                id="review-language"
                value={language}
                onChange={(e) => setLanguage(e.target.value)}
                disabled={mutation.isPending}
                className="w-44"
              >
                {CODE_LANGUAGES.map((l) => (
                  <option key={l} value={l}>
                    {l}
                  </option>
                ))}
              </Select>
            </div>
          )}

          <div className="space-y-1.5">
            <label htmlFor="review-input" className="text-sm font-medium">
              {kind === "code" ? "Your solution" : "Your design (Markdown)"}
            </label>
            <textarea
              id="review-input"
              rows={kind === "code" ? 14 : 10}
              value={text}
              onChange={(e) => setText(e.target.value)}
              disabled={mutation.isPending}
              spellCheck={false}
              placeholder={copy.placeholder}
              className={cn(
                "w-full rounded-md border border-border bg-background px-3 py-2 text-sm outline-none",
                "focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-60",
                kind === "code" && "font-mono leading-relaxed",
              )}
            />
          </div>

          <div className="flex justify-end">
            <Button onClick={() => mutation.mutate()} loading={mutation.isPending} disabled={!canSubmit}>
              {!mutation.isPending && <Wand2 aria-hidden />}
              {copy.cta}
            </Button>
          </div>
        </CardContent>
      </Card>

      {mutation.isPending && <ReviewSkeleton />}
      {mutation.isError && (
        <Alert variant="danger" title="The reviewer couldn't respond">
          The AI service may be busy or unavailable. Try again.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => mutation.mutate()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      )}
      {mutation.data && (
        <Card>
          <CardHeader className="flex-row items-center justify-between gap-2 space-y-0">
            <CardTitle className="text-base">Review</CardTitle>
            {mutation.data.used_fallback ? (
              <Badge variant="secondary" size="sm" title="Generated by a deterministic rubric">
                Heuristic
              </Badge>
            ) : (
              <Badge variant="info" size="sm" title="Generated by the AI model">
                <Sparkles className="size-3" aria-hidden /> AI
              </Badge>
            )}
          </CardHeader>
          <CardContent>
            <Markdown content={mutation.data.content || "_No content returned._"} />
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function ReviewSkeleton() {
  return (
    <Card>
      <CardContent className="space-y-2 py-5" aria-busy>
        <span className="sr-only" role="status">
          Reviewing…
        </span>
        <Skeleton className="h-4 w-1/3" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-5/6" />
        <Skeleton className="h-4 w-2/3" />
      </CardContent>
    </Card>
  );
}
