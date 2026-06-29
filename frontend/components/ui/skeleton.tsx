import * as React from "react";
import { cn } from "@/lib/utils";

/**
 * Loading placeholder (DESIGN-SYSTEM §4 "Skeleton"). Shapes should match the
 * real content they stand in for. `aria-hidden` — a sibling live region should
 * announce "Loading" where relevant.
 */
function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      aria-hidden
      className={cn("animate-pulse rounded-md bg-muted", className)}
      {...props}
    />
  );
}

export { Skeleton };
