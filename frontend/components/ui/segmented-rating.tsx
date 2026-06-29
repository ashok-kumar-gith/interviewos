"use client";

import * as React from "react";
import type { PillarKey } from "@/lib/nav";
import { cn } from "@/lib/utils";

export interface SegmentedRatingProps {
  /** Current value 1–5, or undefined when unset. */
  value?: number;
  onChange: (value: number) => void;
  /** Drives the accent color via `--pillar-${pillar}`; defaults to primary. */
  pillar?: PillarKey;
  /** Accessible group label (the radiogroup name). */
  ariaLabel: string;
  /** Stable id prefix for the segments. */
  name: string;
  min?: number;
  max?: number;
  className?: string;
}

/**
 * A 1–5 segmented control implemented as a WAI-ARIA radio group with roving
 * tabindex and arrow-key navigation. Selected segments fill with the pillar
 * accent. Color is never the sole signal — the chosen value is also surfaced in
 * the parent's label text.
 */
export function SegmentedRating({
  value,
  onChange,
  pillar,
  ariaLabel,
  name,
  min = 1,
  max = 5,
  className,
}: SegmentedRatingProps) {
  const accent = pillar ? `hsl(var(--pillar-${pillar}))` : "hsl(var(--primary))";
  const accentFg = pillar ? "hsl(var(--pillar-foreground))" : "hsl(var(--primary-foreground))";
  const values = React.useMemo(
    () => Array.from({ length: max - min + 1 }, (_, i) => min + i),
    [min, max],
  );

  function handleKeyDown(e: React.KeyboardEvent, v: number) {
    let next: number | null = null;
    if (e.key === "ArrowRight" || e.key === "ArrowUp") next = Math.min(max, (value ?? min) + 1);
    else if (e.key === "ArrowLeft" || e.key === "ArrowDown")
      next = Math.max(min, (value ?? min) - 1);
    else if (e.key === " " || e.key === "Enter") next = v;
    if (next !== null) {
      e.preventDefault();
      onChange(next);
    }
  }

  return (
    <div
      role="radiogroup"
      aria-label={ariaLabel}
      className={cn("grid grid-cols-5 gap-1.5", className)}
    >
      {values.map((v) => {
        const selected = value === v;
        // Roving tabindex: the selected (or first, when unset) segment is tabbable.
        const tabbable = selected || (value === undefined && v === min);
        return (
          <button
            key={v}
            type="button"
            role="radio"
            id={`${name}-${v}`}
            aria-checked={selected}
            tabIndex={tabbable ? 0 : -1}
            onClick={() => onChange(v)}
            onKeyDown={(e) => handleKeyDown(e, v)}
            style={selected ? { backgroundColor: accent, color: accentFg, borderColor: accent } : undefined}
            className={cn(
              "flex h-10 items-center justify-center rounded-md border text-sm font-medium tabular-nums",
              "transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring " +
                "focus-visible:ring-offset-2 focus-visible:ring-offset-background",
              !selected && "border-border bg-background text-foreground hover:bg-muted",
            )}
          >
            {v}
          </button>
        );
      })}
    </div>
  );
}
