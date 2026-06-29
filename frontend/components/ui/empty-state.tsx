import * as React from "react";
import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import { Card } from "@/components/ui/card";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description?: React.ReactNode;
  /** Primary CTA — renders a link Button when `href` is set. */
  actionLabel?: string;
  actionHref?: string;
  className?: string;
  children?: React.ReactNode;
}

/** Centered empty/zero state with an optional primary CTA. */
export function EmptyState({
  icon: Icon,
  title,
  description,
  actionLabel,
  actionHref,
  className,
  children,
}: EmptyStateProps) {
  return (
    <Card className={cn("flex flex-col items-center gap-3 px-6 py-12 text-center", className)}>
      {Icon && (
        <span className="grid size-12 place-items-center rounded-full bg-muted text-muted-foreground">
          <Icon className="size-6" aria-hidden />
        </span>
      )}
      <div className="space-y-1">
        <h2 className="text-h3 font-semibold">{title}</h2>
        {description && (
          <p className="mx-auto max-w-sm text-sm text-muted-foreground">{description}</p>
        )}
      </div>
      {actionLabel && actionHref && (
        <Link href={actionHref} className={cn(buttonVariants(), "mt-2")}>
          {actionLabel}
        </Link>
      )}
      {children}
    </Card>
  );
}
