"use client";

import * as React from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { cn } from "@/lib/utils";
import { useIsAdmin } from "@/lib/store/admin";
import { useAuthStore } from "@/lib/store/auth";
import {
  CONTENT_TYPE_LABEL,
  detailHref,
  type ContentType,
} from "@/components/authoring/shared";
import { describeApiError } from "@/components/authoring/form-shell";
import { ProblemForm } from "@/components/authoring/problem-form";
import { DesignProblemForm } from "@/components/authoring/design-problem-form";
import { LLDProblemForm } from "@/components/authoring/lld-problem-form";
import { TopicForm } from "@/components/authoring/topic-form";
import {
  createProblem,
  createTopic,
  type ProblemWrite,
  type TopicWrite,
} from "@/lib/api/content";
import { createDesignProblem, type DesignProblemWrite } from "@/lib/api/designproblems";
import { createLLDProblem, type LLDProblemWrite } from "@/lib/api/lld";

const TYPES: ContentType[] = ["problem", "design-problem", "lld-problem", "topic"];

function isContentType(v: string | null): v is ContentType {
  return v != null && (TYPES as string[]).includes(v);
}

function AuthoringInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();

  const paramType = searchParams.get("type");
  const [type, setType] = React.useState<ContentType>(
    isContentType(paramType) ? paramType : "problem",
  );

  const [serverError, setServerError] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});

  const reset = React.useCallback(() => {
    setServerError(null);
    setFieldErrors({});
  }, []);

  const onError = React.useCallback((error: unknown) => {
    const { message, fieldErrors } = describeApiError(error);
    setServerError(message);
    setFieldErrors(fieldErrors);
  }, []);

  const onCreated = React.useCallback(
    (createdType: ContentType, id: string) => {
      // Invalidate the relevant catalog list so the new item shows up.
      const listKey: Record<ContentType, string> = {
        problem: "problems",
        "design-problem": "design-problems",
        "lld-problem": "lld-problems",
        topic: "topics",
      };
      void queryClient.invalidateQueries({ queryKey: [listKey[createdType]] });
      router.push(detailHref(createdType, id));
    },
    [queryClient, router],
  );

  const problemMut = useMutation({
    mutationFn: (body: ProblemWrite) => createProblem(body),
    onSuccess: (d) => onCreated("problem", d.id),
    onError,
  });
  const designMut = useMutation({
    mutationFn: (body: DesignProblemWrite) => createDesignProblem(body),
    onSuccess: (d) => onCreated("design-problem", d.id),
    onError,
  });
  const lldMut = useMutation({
    mutationFn: (body: LLDProblemWrite) => createLLDProblem(body),
    onSuccess: (d) => onCreated("lld-problem", d.id),
    onError,
  });
  const topicMut = useMutation({
    mutationFn: (body: TopicWrite) => createTopic(body),
    onSuccess: (d) => onCreated("topic", d.id),
    onError,
  });

  function selectType(next: ContentType) {
    reset();
    setType(next);
    const params = new URLSearchParams(searchParams.toString());
    params.set("type", next);
    router.replace(`/admin/content?${params.toString()}`);
  }

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-h1">New content</h1>
        <p className="text-sm text-muted-foreground">
          Author a problem, design exercise, or topic. Fields marked required must be filled;
          everything else is optional.
        </p>
      </header>

      <div>
        <p className="mb-2 text-sm font-medium">Content type</p>
        <div className="flex flex-wrap gap-2" role="tablist" aria-label="Content type">
          {TYPES.map((t) => (
            <button
              key={t}
              type="button"
              role="tab"
              aria-selected={type === t}
              onClick={() => selectType(t)}
              className={cn(
                "rounded-md border px-3.5 py-1.5 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                type === t
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-border bg-transparent hover:bg-muted",
              )}
            >
              {CONTENT_TYPE_LABEL[t]}
            </button>
          ))}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-h3">{CONTENT_TYPE_LABEL[type]}</CardTitle>
          <CardDescription>
            Submitting creates the item and takes you to its page.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {type === "problem" && (
            <ProblemForm
              submitLabel="Create problem"
              submitting={problemMut.isPending}
              serverError={serverError}
              fieldErrors={fieldErrors}
              onSubmit={(body) => {
                reset();
                problemMut.mutate(body);
              }}
            />
          )}
          {type === "design-problem" && (
            <DesignProblemForm
              submitLabel="Create design problem"
              submitting={designMut.isPending}
              serverError={serverError}
              fieldErrors={fieldErrors}
              onSubmit={(body) => {
                reset();
                designMut.mutate(body);
              }}
            />
          )}
          {type === "lld-problem" && (
            <LLDProblemForm
              submitLabel="Create LLD problem"
              submitting={lldMut.isPending}
              serverError={serverError}
              fieldErrors={fieldErrors}
              onSubmit={(body) => {
                reset();
                lldMut.mutate(body);
              }}
            />
          )}
          {type === "topic" && (
            <TopicForm
              submitLabel="Create topic"
              submitting={topicMut.isPending}
              serverError={serverError}
              fieldErrors={fieldErrors}
              onSubmit={(body) => {
                reset();
                topicMut.mutate(body);
              }}
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default function AuthoringPage() {
  const initialized = useAuthStore((s) => s.initialized);
  const isAdmin = useIsAdmin();

  // Wait for the session check before deciding — avoids flashing the
  // access-denied state during hydration for a logged-in admin.
  if (initialized && !isAdmin) {
    return (
      <div className="mx-auto max-w-lg py-8">
        <Alert variant="danger" title="Admin access required">
          Content authoring is available to admins only. If you believe this is a mistake, contact
          your workspace owner.
        </Alert>
      </div>
    );
  }

  return (
    <React.Suspense fallback={null}>
      <AuthoringInner />
    </React.Suspense>
  );
}
