"use client";

import * as React from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowUpRight, Brain, Check, RefreshCw, X } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Button, buttonVariants } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import {
  getDueRevisions,
  recordRecall,
  type RecallOutcome,
  type RevisionItem,
} from "@/lib/api/revision";
import { pillarLabel } from "@/lib/pillar-meta";
import { taskItemHref, itemTypeLabel } from "@/lib/task-href";
import type { PlanItemType } from "@/lib/api/types";
import { cn } from "@/lib/utils";

const REVISION_DUE_KEY = ["revision", "due"] as const;
const DASHBOARD_KEY = ["dashboard"] as const;

export default function RevisionPage() {
  const queryClient = useQueryClient();

  const query = useQuery<RevisionItem[], unknown>({
    queryKey: REVISION_DUE_KEY,
    queryFn: () => getDueRevisions(),
  });

  const recallMutation = useMutation({
    mutationFn: ({ id, recall }: { id: string; recall: RecallOutcome }) =>
      recordRecall(id, recall, 0),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: REVISION_DUE_KEY });
      void queryClient.invalidateQueries({ queryKey: DASHBOARD_KEY });
    },
  });

  const items = query.data ?? [];
  const dueCount = items.length;

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <header>
        <h1 className="text-h1">Revision</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Spaced repetition keeps what you&apos;ve learned from fading.{" "}
          {!query.isLoading && !query.isError && (
            <span className="font-medium text-foreground">
              {dueCount} {dueCount === 1 ? "item" : "items"} due
            </span>
          )}
        </p>
      </header>

      {query.isLoading ? (
        <div className="space-y-3" aria-busy>
          <span className="sr-only" role="status">
            Loading items due for revision
          </span>
          {[0, 1, 2, 3].map((i) => (
            <Card key={i} className="p-5">
              <Skeleton className="h-5 w-1/2" />
              <Skeleton className="mt-2 h-4 w-1/3" />
            </Card>
          ))}
        </div>
      ) : query.isError ? (
        <Alert variant="danger" title="Couldn't load your revision queue">
          Something went wrong.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => query.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      ) : items.length === 0 ? (
        <EmptyState
          icon={Brain}
          title="All caught up — no items due for revision"
          description="Items reappear here as they become due. Keep up the good work."
        />
      ) : (
        <ul className="space-y-2">
          {items.map((item) => {
            const pending =
              recallMutation.isPending && recallMutation.variables?.id === item.id;
            // Resolve the in-app detail route for the underlying item so the user
            // can open the actual question/resource (with its external link) before
            // grading recall — reuses the same polymorphic mapping as plan tasks.
            const href = taskItemHref(item.item_type as PlanItemType, item.item_id);
            return (
              <li key={item.id}>
                <Card>
                  <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
                    <div className="min-w-0 flex-1">
                      {href ? (
                        <Link
                          href={href}
                          className="inline-flex items-center gap-1 font-medium leading-snug hover:text-primary hover:underline"
                        >
                          {itemTitle(item)}
                          <ArrowUpRight className="size-3.5 opacity-60" aria-hidden />
                        </Link>
                      ) : (
                        <p className="font-medium leading-snug">{itemTitle(item)}</p>
                      )}
                      <p className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                        <span className="rounded bg-muted px-1.5 py-0.5 text-2xs uppercase tracking-wide">
                          {itemTypeLabel(item.item_type as PlanItemType)}
                        </span>
                        <span>{pillarLabel(item.pillar_type)}</span>
                        <span aria-hidden>&middot;</span>
                        <span>
                          Stage {item.stage} &middot; every {item.interval_days}{" "}
                          {item.interval_days === 1 ? "day" : "days"}
                        </span>
                        <span aria-hidden>&middot;</span>
                        <span>
                          {item.review_count}{" "}
                          {item.review_count === 1 ? "review" : "reviews"}
                        </span>
                      </p>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      {href && (
                        <Link
                          href={href}
                          className={cn(buttonVariants({ variant: "ghost", size: "sm" }))}
                          aria-label={`Review ${itemTitle(item)}`}
                        >
                          Review
                        </Link>
                      )}
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          recallMutation.mutate({ id: item.id, recall: "incorrect" })
                        }
                        disabled={pending}
                        loading={
                          pending && recallMutation.variables?.recall === "incorrect"
                        }
                      >
                        <X aria-hidden /> Forgot
                      </Button>
                      <Button
                        size="sm"
                        onClick={() =>
                          recallMutation.mutate({ id: item.id, recall: "correct" })
                        }
                        disabled={pending}
                        loading={
                          pending && recallMutation.variables?.recall === "correct"
                        }
                      >
                        <Check aria-hidden /> Got it
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

/** Use the backend title when present; otherwise humanize item_type + pillar. */
function itemTitle(item: RevisionItem): string {
  const title = item.title?.trim();
  if (title) return title;
  const kind = humanize(item.item_type);
  return `${kind} · ${pillarLabel(item.pillar_type)}`;
}

function humanize(value: string): string {
  if (!value) return "Item";
  return value
    .split(/[_\s-]+/)
    .filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}
