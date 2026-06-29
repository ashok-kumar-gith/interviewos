"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Dialog } from "@/components/ui/dialog";
import { Kbd } from "@/components/ui/kbd";
import { useCommandStore } from "@/lib/store/command";
import { GO_TO_MAP, SHORTCUT_GROUPS } from "@/lib/shortcuts";

/** True when focus is in a field where typing should suppress shortcuts. */
function isTypingTarget(el: EventTarget | null): boolean {
  if (!(el instanceof HTMLElement)) return false;
  const tag = el.tagName;
  return (
    tag === "INPUT" ||
    tag === "TEXTAREA" ||
    tag === "SELECT" ||
    el.isContentEditable
  );
}

/**
 * Global keyboard shortcuts (DESIGN-SYSTEM §6). Mounts a window key listener and
 * the `?` help dialog. Bindings are suppressed while typing (except ⌘K / Esc).
 * `g`-prefixed chords navigate; `/` focuses search; `?` opens help.
 */
export function KeyboardShortcuts() {
  const router = useRouter();
  const openPalette = useCommandStore((s) => s.openPalette);
  const helpOpen = useCommandStore((s) => s.helpOpen);
  const openHelp = useCommandStore((s) => s.openHelp);
  const closeHelp = useCommandStore((s) => s.closeHelp);

  // 1s chord window after pressing `g`.
  const chordRef = React.useRef<{ key: string; at: number } | null>(null);

  React.useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      const mod = e.metaKey || e.ctrlKey;

      // ⌘K / Ctrl+K — works even while typing.
      if (mod && (e.key === "k" || e.key === "K")) {
        e.preventDefault();
        openPalette();
        return;
      }

      if (mod || e.altKey) return; // leave other modifier combos to the browser
      if (isTypingTarget(e.target)) return; // suppress while typing

      // Active chord? (e.g. `g` then `d`)
      const chord = chordRef.current;
      if (chord && Date.now() - chord.at < 1000) {
        chordRef.current = null;
        if (chord.key === "g") {
          const href = GO_TO_MAP[e.key.toLowerCase()];
          if (href) {
            e.preventDefault();
            router.push(href);
            return;
          }
        }
      }

      if (e.key === "g") {
        chordRef.current = { key: "g", at: Date.now() };
        return;
      }
      if (e.key === "/") {
        e.preventDefault();
        // Prefer focusing a visible search box; fall back to the palette.
        const search = document.querySelector<HTMLElement>("[data-global-search]");
        if (search) search.focus();
        else openPalette();
        return;
      }
      if (e.key === "?") {
        e.preventDefault();
        openHelp();
        return;
      }
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [router, openPalette, openHelp]);

  return (
    <Dialog
      open={helpOpen}
      onClose={closeHelp}
      title="Keyboard shortcuts"
      description="InterviewOS is keyboard-driven. Press ⌘K anytime to jump anywhere."
    >
      <div className="grid gap-6 sm:grid-cols-2">
        {SHORTCUT_GROUPS.map((group) => (
          <section key={group.title} aria-labelledby={`shortcuts-${group.title}`}>
            <h3
              id={`shortcuts-${group.title}`}
              className="mb-2 text-2xs uppercase text-muted-foreground"
            >
              {group.title}
            </h3>
            <ul className="space-y-1.5">
              {group.rows.map((row, i) => (
                <li key={i} className="flex items-center justify-between gap-3 text-sm">
                  <span className="text-foreground/90">{row.description}</span>
                  <Kbd keys={row.keys} />
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
    </Dialog>
  );
}
