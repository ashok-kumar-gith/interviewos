import * as React from "react";
import { cn } from "@/lib/utils";

export interface ProgressRingProps {
  /** 0–100. */
  value: number;
  /** Outer diameter in px. */
  size?: number;
  /** Stroke width in px. */
  stroke?: number;
  /** CSS color for the progress arc (e.g. "hsl(var(--primary))"). */
  color?: string;
  /** Accessible label, e.g. "Overall readiness 68 percent". */
  ariaLabel: string;
  className?: string;
  /** Whether to render the center percentage label. */
  showLabel?: boolean;
}

/**
 * Compact circular progress indicator (DESIGN-SYSTEM §3 "Progress ring").
 * Static (CSS) sweep — celebratory animation is layered by callers.
 */
export function ProgressRing({
  value,
  size = 64,
  stroke = 6,
  color = "hsl(var(--primary))",
  ariaLabel,
  className,
  showLabel = true,
}: ProgressRingProps) {
  const clamped = Math.max(0, Math.min(100, value));
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (clamped / 100) * circumference;

  return (
    <div
      role="img"
      aria-label={ariaLabel}
      className={cn("relative shrink-0", className)}
      style={{ width: size, height: size }}
    >
      <svg width={size} height={size} className="-rotate-90">
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="hsl(var(--muted))"
          strokeWidth={stroke}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={color}
          strokeWidth={stroke}
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          strokeLinecap="round"
        />
      </svg>
      {showLabel && (
        <span
          className="absolute inset-0 grid place-items-center text-sm font-semibold tabular-nums"
          aria-hidden
        >
          {Math.round(clamped)}%
        </span>
      )}
    </div>
  );
}
