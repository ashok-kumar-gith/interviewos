"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowLeft, RefreshCw, type LucideIcon } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { Markdown } from "@/components/ui/markdown";
import { cn } from "@/lib/utils";

/** Back-to-catalog link rendered above the detail header. */
export function BackLink({ href, label }: { href: string; label: string }) {
  return (
    <Link
      href={href}
      className="inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
    >
      <ArrowLeft className="size-4" aria-hidden />
      {label}
    </Link>
  );
}

/** A titled content card used to group a section of a detail page. */
export function DetailSection({
  title,
  icon: Icon,
  children,
  className,
}: {
  title: string;
  icon?: LucideIcon;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <Card className={cn("p-5", className)}>
      <h2 className="mb-3 flex items-center gap-2 text-h3 font-semibold">
        {Icon && <Icon className="size-4 text-muted-foreground" aria-hidden />}
        {title}
      </h2>
      {children}
    </Card>
  );
}

/** Render a markdown section only when content is present. */
export function MarkdownSection({
  title,
  icon,
  content,
}: {
  title: string;
  icon?: LucideIcon;
  content?: string | null;
}) {
  if (!content || content.trim() === "") return null;
  return (
    <DetailSection title={title} icon={icon}>
      <Markdown content={content} />
    </DetailSection>
  );
}

/** Loading skeleton shared by all detail pages. */
export function DetailSkeleton() {
  return (
    <div className="space-y-6" aria-busy>
      <span className="sr-only" role="status">
        Loading
      </span>
      <Skeleton className="h-4 w-32" />
      <div className="space-y-3">
        <Skeleton className="h-8 w-2/3" />
        <Skeleton className="h-4 w-40" />
      </div>
      {[0, 1, 2].map((i) => (
        <Skeleton key={i} className="h-32 w-full" />
      ))}
    </div>
  );
}

/** Error / not-found states for a detail page. */
export function DetailError({
  notFound,
  backHref,
  backLabel,
  onRetry,
}: {
  notFound: boolean;
  backHref: string;
  backLabel: string;
  onRetry: () => void;
}) {
  if (notFound) {
    return (
      <div className="space-y-6">
        <BackLink href={backHref} label={backLabel} />
        <EmptyState
          title="Not found"
          description="We couldn't find what you were looking for. It may have been moved or removed."
          actionLabel={backLabel}
          actionHref={backHref}
        />
      </div>
    );
  }
  return (
    <div className="space-y-6">
      <BackLink href={backHref} label={backLabel} />
      <Alert variant="danger" title="Couldn't load this page">
        Something went wrong. Try again.
        <div className="mt-3">
          <Button variant="outline" size="sm" onClick={onRetry}>
            <RefreshCw aria-hidden /> Retry
          </Button>
        </div>
      </Alert>
    </div>
  );
}
