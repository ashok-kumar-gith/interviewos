/**
 * Shortcut + command registry. Plain data shared by the command palette (⌘K),
 * the keyboard-shortcuts help dialog (?), and the global key handler. Mirrors the
 * bindings in DESIGN-SYSTEM §6. No JSX here so it stays import-safe everywhere.
 */

export interface ShortcutRow {
  /** Tokens to render as <kbd> (a chord is multiple tokens). */
  keys: string[];
  description: string;
}

export interface ShortcutGroup {
  title: string;
  rows: ShortcutRow[];
}

/** The full shortcut cheat-sheet, grouped for the `?` help dialog. */
export const SHORTCUT_GROUPS: ShortcutGroup[] = [
  {
    title: "Global",
    rows: [
      { keys: ["⌘", "K"], description: "Open command palette" },
      { keys: ["/"], description: "Focus search" },
      { keys: ["?"], description: "Open this help" },
      { keys: ["Esc"], description: "Close dialog, sheet, or palette" },
    ],
  },
  {
    title: "Go to",
    rows: [
      { keys: ["g", "d"], description: "Dashboard" },
      { keys: ["g", "t"], description: "Today" },
      { keys: ["g", "r"], description: "Roadmap" },
      { keys: ["g", "p"], description: "Problems (DSA)" },
      { keys: ["g", "b"], description: "Behavioral" },
      { keys: ["g", "a"], description: "Analytics" },
      { keys: ["g", "v"], description: "Revision" },
      { keys: ["g", "s"], description: "Settings" },
    ],
  },
  {
    title: "Lists & tasks",
    rows: [
      { keys: ["j"], description: "Move focus down" },
      { keys: ["k"], description: "Move focus up" },
      { keys: ["c"], description: "Complete focused task" },
      { keys: ["e"], description: "Edit focused task" },
    ],
  },
];

/** Chord navigation map: g-prefixed go-to destinations. */
export const GO_TO_MAP: Record<string, string> = {
  d: "/dashboard",
  t: "/today",
  r: "/roadmap",
  p: "/problems",
  b: "/behavioral",
  a: "/analytics",
  s: "/settings",
  v: "/revision",
};

export type CommandKind = "navigate" | "action";

export interface CommandItem {
  id: string;
  /** Display label, used for fuzzy matching. */
  label: string;
  /** Extra search keywords. */
  keywords?: string;
  group: string;
  kind: CommandKind;
  /** For navigate items. */
  href?: string;
  /** kbd hint tokens, when a binding exists. */
  shortcut?: string[];
}

/** Navigation entries surfaced in the palette. */
export const NAV_COMMANDS: CommandItem[] = [
  { id: "nav-dashboard", label: "Dashboard", group: "Navigate", kind: "navigate", href: "/dashboard", shortcut: ["g", "d"], keywords: "home readiness" },
  { id: "nav-today", label: "Today", group: "Navigate", kind: "navigate", href: "/today", shortcut: ["g", "t"], keywords: "tasks plan" },
  { id: "nav-roadmap", label: "Roadmap", group: "Navigate", kind: "navigate", href: "/roadmap", shortcut: ["g", "r"], keywords: "plan weeks timeline" },
  { id: "nav-analytics", label: "Analytics", group: "Navigate", kind: "navigate", href: "/analytics", shortcut: ["g", "a"], keywords: "progress readiness trend charts time spent topics" },
  { id: "nav-problems", label: "Problems", group: "Navigate", kind: "navigate", href: "/problems", shortcut: ["g", "p"], keywords: "dsa leetcode coding" },
  { id: "nav-system-design", label: "System Design", group: "Navigate", kind: "navigate", href: "/system-design", keywords: "hld architecture" },
  { id: "nav-lld", label: "LLD", group: "Navigate", kind: "navigate", href: "/lld", keywords: "low level design oop patterns" },
  { id: "nav-backend", label: "Backend Engineering", group: "Navigate", kind: "navigate", href: "/backend-engineering", keywords: "backend depth" },
  { id: "nav-behavioral", label: "Behavioral", group: "Navigate", kind: "navigate", href: "/behavioral", shortcut: ["g", "b"], keywords: "star stories" },
  { id: "nav-resume", label: "Resume", group: "Navigate", kind: "navigate", href: "/resume", keywords: "ats score cv" },
  { id: "nav-coach", label: "AI Coach", group: "Navigate", kind: "navigate", href: "/coach", keywords: "ai assistant ask question help chat" },
  { id: "nav-resources", label: "Resource Library", group: "Navigate", kind: "navigate", href: "/resources", keywords: "books videos articles courses" },
  { id: "nav-revision", label: "Revision", group: "Navigate", kind: "navigate", href: "/revision", shortcut: ["g", "v"], keywords: "spaced repetition recall review due" },
  { id: "nav-mock", label: "Mock Interviews", group: "Navigate", kind: "navigate", href: "/mock", keywords: "practice interview" },
  { id: "nav-settings", label: "Settings", group: "Navigate", kind: "navigate", href: "/settings", shortcut: ["g", "s"], keywords: "preferences account" },
];

/** Quick actions (route-based shortcuts to common flows). */
export const ACTION_COMMANDS: CommandItem[] = [
  { id: "action-roadmap", label: "Generate roadmap", group: "Actions", kind: "navigate", href: "/intake", keywords: "plan create intake start" },
  { id: "action-log-mock", label: "Log a mock", group: "Actions", kind: "navigate", href: "/mock", keywords: "record interview result" },
  { id: "action-today", label: "Go to today's tasks", group: "Actions", kind: "navigate", href: "/today", keywords: "plan tasks" },
  { id: "action-ask-coach", label: "Ask the AI coach", group: "Actions", kind: "navigate", href: "/coach", keywords: "ai help question assistant" },
  { id: "action-review-due", label: "Review due items", group: "Actions", kind: "navigate", href: "/revision", keywords: "spaced repetition recall flashcards" },
];

export const ALL_COMMANDS: CommandItem[] = [...NAV_COMMANDS, ...ACTION_COMMANDS];

/**
 * Tiny subsequence fuzzy matcher. Returns a score (higher = better) or -1 when
 * the query is not a subsequence of the haystack. Consecutive and word-start
 * matches score higher. No deps.
 */
export function fuzzyScore(query: string, target: string): number {
  const q = query.trim().toLowerCase();
  if (q === "") return 0;
  const t = target.toLowerCase();
  let score = 0;
  let ti = 0;
  let streak = 0;
  for (let qi = 0; qi < q.length; qi++) {
    const ch = q[qi];
    const found = t.indexOf(ch, ti);
    if (found === -1) return -1;
    // Bonus for matching at a word boundary or start.
    if (found === 0 || t[found - 1] === " " || t[found - 1] === "-") score += 6;
    // Bonus for consecutive matches.
    if (found === ti) {
      streak += 1;
      score += 2 + streak;
    } else {
      streak = 0;
    }
    score += 1;
    ti = found + 1;
  }
  // Prefer shorter targets.
  score -= Math.max(0, t.length - q.length) * 0.05;
  return score;
}

/** Filter + rank commands against a query (matches label + keywords). */
export function filterCommands(commands: CommandItem[], query: string): CommandItem[] {
  if (query.trim() === "") return commands;
  return commands
    .map((cmd) => {
      const labelScore = fuzzyScore(query, cmd.label);
      const kwScore = cmd.keywords ? fuzzyScore(query, cmd.keywords) : -1;
      const score = Math.max(labelScore, kwScore === -1 ? -1 : kwScore - 2);
      return { cmd, score };
    })
    .filter((r) => r.score >= 0)
    .sort((a, b) => b.score - a.score)
    .map((r) => r.cmd);
}
