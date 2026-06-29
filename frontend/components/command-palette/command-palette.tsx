"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { CornerDownLeft, Search } from "lucide-react";

import { Kbd } from "@/components/ui/kbd";
import { useCommandStore } from "@/lib/store/command";
import {
  ALL_COMMANDS,
  filterCommands,
  type CommandItem,
} from "@/lib/shortcuts";
import { cn } from "@/lib/utils";

/**
 * Global command palette (⌘K). Fuzzy-filterable list of navigation + quick
 * actions, with arrow-key navigation and Enter to run. Opened via the topbar
 * search button or the global key handler; controlled through the command store.
 */
export function CommandPalette() {
  const router = useRouter();
  const open = useCommandStore((s) => s.paletteOpen);
  const close = useCommandStore((s) => s.closePalette);

  const [query, setQuery] = React.useState("");
  const [active, setActive] = React.useState(0);
  const inputRef = React.useRef<HTMLInputElement>(null);
  const listRef = React.useRef<HTMLUListElement>(null);
  const restoreRef = React.useRef<HTMLElement | null>(null);
  const titleId = React.useId();

  const results = React.useMemo(() => filterCommands(ALL_COMMANDS, query), [query]);

  // Reset state and capture focus on open; restore focus on close.
  React.useEffect(() => {
    if (!open) return;
    restoreRef.current = document.activeElement as HTMLElement | null;
    setQuery("");
    setActive(0);
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const t = window.setTimeout(() => inputRef.current?.focus(), 0);
    return () => {
      window.clearTimeout(t);
      document.body.style.overflow = prevOverflow;
      restoreRef.current?.focus?.();
    };
  }, [open]);

  // Clamp active index when the result set shrinks.
  React.useEffect(() => {
    setActive((a) => (a >= results.length ? 0 : a));
  }, [results.length]);

  // Keep the active item scrolled into view.
  React.useEffect(() => {
    if (!open) return;
    const el = listRef.current?.querySelector<HTMLElement>(`[data-index="${active}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [active, open]);

  const run = React.useCallback(
    (cmd: CommandItem | undefined) => {
      if (!cmd) return;
      close();
      if (cmd.href) router.push(cmd.href);
    },
    [close, router],
  );

  function onKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Escape") {
      e.preventDefault();
      close();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((a) => (results.length ? (a + 1) % results.length : 0));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((a) => (results.length ? (a - 1 + results.length) % results.length : 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      run(results[active]);
    }
  }

  if (!open) return null;

  // Group results in stable group order while preserving rank within a group.
  const groups: { title: string; items: { cmd: CommandItem; index: number }[] }[] = [];
  results.forEach((cmd, index) => {
    let g = groups.find((x) => x.title === cmd.group);
    if (!g) {
      g = { title: cmd.group, items: [] };
      groups.push(g);
    }
    g.items.push({ cmd, index });
  });

  return (
    <div className="fixed inset-0 z-modal flex items-start justify-center overflow-y-auto p-4 pt-[12vh]">
      <div
        className="fixed inset-0 z-overlay animate-fade-in bg-background/70 backdrop-blur-sm"
        onClick={close}
        aria-hidden
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        aria-labelledby={titleId}
        onKeyDown={onKeyDown}
        className="relative z-modal w-full max-w-xl animate-scale-in overflow-hidden rounded-xl border border-border bg-popover text-popover-foreground shadow-elevation-3"
      >
        <h2 id={titleId} className="sr-only">
          Command palette
        </h2>
        <div className="flex items-center gap-2 border-b border-border px-3.5">
          <Search className="size-4 shrink-0 text-muted-foreground" aria-hidden />
          <input
            ref={inputRef}
            type="text"
            role="combobox"
            aria-expanded
            aria-controls="command-list"
            aria-activedescendant={results[active] ? `command-${results[active].id}` : undefined}
            placeholder="Search actions and pages…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="h-12 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
          <Kbd>Esc</Kbd>
        </div>

        <ul
          id="command-list"
          ref={listRef}
          role="listbox"
          aria-label="Commands"
          className="max-h-[50vh] overflow-y-auto p-2"
        >
          {results.length === 0 ? (
            <li className="px-3 py-6 text-center text-sm text-muted-foreground">
              No commands match “{query}”.
            </li>
          ) : (
            groups.map((g) => (
              <li key={g.title}>
                <p className="px-2 pb-1 pt-2 text-2xs uppercase text-muted-foreground">
                  {g.title}
                </p>
                <ul>
                  {g.items.map(({ cmd, index }) => (
                    <li
                      key={cmd.id}
                      id={`command-${cmd.id}`}
                      role="option"
                      aria-selected={index === active}
                      data-index={index}
                      onMouseEnter={() => setActive(index)}
                      onClick={() => run(cmd)}
                      className={cn(
                        "flex cursor-pointer items-center gap-2 rounded-md px-2 py-2 text-sm",
                        index === active ? "bg-muted text-foreground" : "text-foreground/90",
                      )}
                    >
                      <span className="min-w-0 flex-1 truncate">{cmd.label}</span>
                      {cmd.shortcut && <Kbd keys={cmd.shortcut} />}
                      {index === active && (
                        <CornerDownLeft
                          className="size-3.5 text-muted-foreground"
                          aria-hidden
                        />
                      )}
                    </li>
                  ))}
                </ul>
              </li>
            ))
          )}
        </ul>
      </div>
    </div>
  );
}
