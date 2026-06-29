import * as React from "react";
import { cn } from "@/lib/utils";

/**
 * Keyboard hint. Renders one or more keys; pass a `keys` array for a chord
 * (e.g. ["g", "d"]) or children for a single token. Decorative — it mirrors a
 * real binding (DESIGN-SYSTEM §3, §6).
 */
export function Kbd({
  children,
  keys,
  className,
}: {
  children?: React.ReactNode;
  keys?: React.ReactNode[];
  className?: string;
}) {
  if (keys && keys.length > 0) {
    return (
      <span className="inline-flex items-center gap-1">
        {keys.map((k, i) => (
          <Kbd key={i} className={className}>
            {k}
          </Kbd>
        ))}
      </span>
    );
  }
  return (
    <kbd
      className={cn(
        "inline-flex h-5 min-w-5 items-center justify-center gap-0.5 rounded-sm border border-border " +
          "bg-muted px-1.5 font-mono text-2xs font-medium text-muted-foreground",
        className,
      )}
    >
      {children}
    </kbd>
  );
}
