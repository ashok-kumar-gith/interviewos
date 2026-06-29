"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

export interface TooltipProps {
  /** The tooltip text/content. */
  content: React.ReactNode;
  children: React.ReactNode;
  /** Side the tooltip appears on. */
  side?: "top" | "bottom";
  className?: string;
}

/**
 * Lightweight hover/focus tooltip (no Radix). Opens on pointer-enter / focus
 * (400ms open delay per DESIGN-SYSTEM §3), closes immediately on leave/blur.
 * Never the sole carrier of critical info.
 */
export function Tooltip({ content, children, side = "top", className }: TooltipProps) {
  const [open, setOpen] = React.useState(false);
  const timer = React.useRef<number | undefined>(undefined);
  const id = React.useId();

  const show = React.useCallback(() => {
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => setOpen(true), 400);
  }, []);
  const hide = React.useCallback(() => {
    window.clearTimeout(timer.current);
    setOpen(false);
  }, []);

  React.useEffect(() => () => window.clearTimeout(timer.current), []);

  return (
    <span
      className="relative inline-flex"
      onPointerEnter={show}
      onPointerLeave={hide}
      onFocus={show}
      onBlur={hide}
      aria-describedby={open ? id : undefined}
    >
      {children}
      {open && (
        <span
          role="tooltip"
          id={id}
          className={cn(
            "pointer-events-none absolute left-1/2 z-dropdown -translate-x-1/2 animate-fade-in whitespace-nowrap rounded-md " +
              "border border-border bg-popover px-2 py-1 text-xs text-popover-foreground shadow-elevation-2",
            side === "top" ? "bottom-full mb-1.5" : "top-full mt-1.5",
            className,
          )}
        >
          {content}
        </span>
      )}
    </span>
  );
}
