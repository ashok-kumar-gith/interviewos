import * as React from "react";
import { Check } from "lucide-react";
import { cn } from "@/lib/utils";

export interface StepperStep {
  /** Short label shown under the node on wider screens. */
  label: string;
}

export interface StepperProps {
  steps: StepperStep[];
  /** Zero-based index of the active step. */
  current: number;
}

/**
 * Horizontal progress indicator for the intake wizard. Each node is complete
 * (filled + check), current (ringed), or upcoming (muted). Conveys progress as
 * "Step N of M" to assistive tech via the wrapping region label.
 */
export function Stepper({ steps, current }: StepperProps) {
  return (
    <ol
      className="flex items-center"
      aria-label={`Step ${current + 1} of ${steps.length}`}
    >
      {steps.map((step, i) => {
        const isComplete = i < current;
        const isCurrent = i === current;
        const isLast = i === steps.length - 1;
        return (
          <li
            key={step.label}
            className={cn("flex items-center", !isLast && "flex-1")}
            aria-current={isCurrent ? "step" : undefined}
          >
            <div className="flex flex-col items-center gap-1.5">
              <span
                className={cn(
                  "grid size-7 shrink-0 place-items-center rounded-full border text-xs font-semibold tabular-nums transition-colors",
                  isComplete && "border-primary bg-primary text-primary-foreground",
                  isCurrent &&
                    "border-primary bg-background text-primary shadow-glow-primary",
                  !isComplete && !isCurrent && "border-border bg-muted text-muted-foreground",
                )}
              >
                {isComplete ? <Check className="size-4" aria-hidden /> : i + 1}
              </span>
              <span
                className={cn(
                  "hidden text-2xs uppercase tracking-wide sm:block",
                  isCurrent ? "text-foreground" : "text-muted-foreground",
                )}
              >
                {step.label}
              </span>
            </div>
            {!isLast && (
              <span
                aria-hidden
                className={cn(
                  "mx-2 h-px flex-1 transition-colors",
                  isComplete ? "bg-primary" : "bg-border",
                )}
              />
            )}
          </li>
        );
      })}
    </ol>
  );
}
