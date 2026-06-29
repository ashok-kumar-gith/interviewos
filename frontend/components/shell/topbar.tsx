"use client";

import { Command, Flame, Moon, PanelLeft, Search, Sun, User } from "lucide-react";
import { Button } from "@/components/ui/button";
import { NotificationBell } from "@/components/shell/notification-bell";
import { useUiStore } from "@/lib/store/ui";
import { useCommandStore } from "@/lib/store/command";

export function Topbar() {
  const toggleSidebar = useUiStore((s) => s.toggleSidebar);
  const theme = useUiStore((s) => s.theme);
  const toggleTheme = useUiStore((s) => s.toggleTheme);
  const openPalette = useCommandStore((s) => s.openPalette);

  return (
    <header className="sticky top-0 z-nav flex h-14 items-center gap-3 border-b border-border bg-surface px-4">
      <Button
        variant="ghost"
        size="icon"
        className="size-8"
        onClick={toggleSidebar}
        aria-label="Toggle sidebar"
      >
        <PanelLeft className="size-4" />
      </Button>

      <span className="text-sm font-semibold tracking-tight">InterviewOS</span>

      {/* Centered search / command-palette trigger */}
      <div className="flex flex-1 justify-center px-2">
        <button
          type="button"
          data-global-search
          onClick={openPalette}
          className="flex h-8 w-full max-w-md items-center gap-2 rounded-md border border-border bg-background px-3 text-sm text-muted-foreground transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          aria-label="Search — open command palette"
        >
          <Search className="size-4 shrink-0" aria-hidden />
          <span className="truncate">Search…</span>
          <kbd className="ml-auto flex items-center gap-0.5 rounded-sm border border-border bg-muted px-1.5 py-0.5 text-2xs font-medium text-muted-foreground">
            <Command className="size-3" aria-hidden />K
          </kbd>
        </button>
      </div>

      {/* Streak chip (placeholder) */}
      <span
        className="flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-1 text-xs font-medium tabular-nums"
        aria-label="12 day streak"
      >
        <Flame className="size-3.5 text-warning" aria-hidden />
        12d
      </span>

      <NotificationBell />

      <Button
        variant="ghost"
        size="icon"
        className="size-8"
        onClick={toggleTheme}
        aria-label="Toggle theme"
      >
        {theme === "dark" ? <Sun className="size-4" /> : <Moon className="size-4" />}
      </Button>

      <Button variant="ghost" size="icon" className="size-8" aria-label="Profile menu">
        <User className="size-4" />
      </Button>
    </header>
  );
}
