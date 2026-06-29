"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

export interface TabItem {
  value: string;
  label: React.ReactNode;
}

export interface TabsProps {
  tabs: TabItem[];
  value: string;
  onValueChange: (value: string) => void;
  /** Accessible label for the tablist. */
  ariaLabel: string;
  className?: string;
}

/**
 * Underline-style tab bar implemented as a WAI-ARIA tablist with roving
 * tabindex + arrow-key navigation. The caller renders the matching panel with
 * `role="tabpanel"` and `aria-labelledby={tabId(value)}`.
 */
export function Tabs({ tabs, value, onValueChange, ariaLabel, className }: TabsProps) {
  const idBase = React.useId();
  const tabId = (v: string) => `${idBase}-tab-${v}`;

  function onKeyDown(e: React.KeyboardEvent) {
    const idx = tabs.findIndex((t) => t.value === value);
    let next = -1;
    if (e.key === "ArrowRight") next = (idx + 1) % tabs.length;
    else if (e.key === "ArrowLeft") next = (idx - 1 + tabs.length) % tabs.length;
    else if (e.key === "Home") next = 0;
    else if (e.key === "End") next = tabs.length - 1;
    if (next >= 0) {
      e.preventDefault();
      const nextValue = tabs[next].value;
      onValueChange(nextValue);
      document.getElementById(tabId(nextValue))?.focus();
    }
  }

  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      onKeyDown={onKeyDown}
      className={cn("flex gap-1 border-b border-border", className)}
    >
      {tabs.map((t) => {
        const selected = t.value === value;
        return (
          <button
            key={t.value}
            type="button"
            role="tab"
            id={tabId(t.value)}
            aria-selected={selected}
            tabIndex={selected ? 0 : -1}
            onClick={() => onValueChange(t.value)}
            className={cn(
              "relative -mb-px flex items-center gap-1.5 border-b-2 px-3 py-2 text-sm font-medium",
              "transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background",
              selected
                ? "border-primary text-foreground"
                : "border-transparent text-muted-foreground hover:text-foreground",
            )}
          >
            {t.label}
          </button>
        );
      })}
    </div>
  );
}

/** Build the tab id so a panel can reference it via aria-labelledby. */
export function makeTabId(idBase: string, value: string): string {
  return `${idBase}-tab-${value}`;
}
