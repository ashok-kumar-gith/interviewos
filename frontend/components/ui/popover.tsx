"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

export interface PopoverProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The trigger element (rendered inline; must forward a ref + onClick). */
  trigger: React.ReactNode;
  children: React.ReactNode;
  /** Panel alignment relative to the trigger. */
  align?: "start" | "end";
  className?: string;
  /** Accessible label for the floating panel. */
  ariaLabel?: string;
}

/**
 * Lightweight floating panel (no Radix). Click-outside and Esc dismiss; the
 * caller supplies the trigger. Panel is positioned absolutely under the trigger.
 */
export function Popover({
  open,
  onOpenChange,
  trigger,
  children,
  align = "end",
  className,
  ariaLabel,
}: PopoverProps) {
  const rootRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    if (!open) return;
    function onDocClick(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) onOpenChange(false);
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onOpenChange(false);
    }
    document.addEventListener("mousedown", onDocClick);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onDocClick);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open, onOpenChange]);

  return (
    <div ref={rootRef} className="relative">
      {trigger}
      {open && (
        <div
          role="dialog"
          aria-label={ariaLabel}
          className={cn(
            "absolute top-full z-dropdown mt-2 animate-scale-in rounded-lg border border-border bg-popover text-popover-foreground shadow-elevation-2",
            align === "end" ? "right-0" : "left-0",
            className,
          )}
        >
          {children}
        </div>
      )}
    </div>
  );
}
