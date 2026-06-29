# InterviewOS — Frontend Design System

**Status:** v1.0
**Owner:** Product Design / Frontend Engineering
**Last updated:** 2026-06-29
**Stack:** React 19 · Next.js (App Router) · TypeScript · TailwindCSS · shadcn/ui (Radix) · Recharts · Framer Motion · lucide-react

> This is the canonical source of truth for how InterviewOS looks, feels, and behaves
> on the frontend. It pairs with the PRD (`01-PRD.md`). Engineers and designers should
> treat every token, component contract, and interaction rule here as binding. No
> ad-hoc colors, no one-off spacing, no surprise animations.

---

## Table of contents

1. [Design principles](#1-design-principles)
2. [Design tokens](#2-design-tokens)
3. [Component inventory (primitives)](#3-component-inventory-primitives)
4. [App-specific components](#4-app-specific-components)
5. [Layout system](#5-layout-system)
6. [Keyboard shortcuts & interaction model](#6-keyboard-shortcuts--interaction-model)
7. [Motion & animation](#7-motion--animation)
8. [Accessibility (WCAG 2.1 AA)](#8-accessibility-wcag-21-aa)
9. [Iconography & illustration](#9-iconography--illustration)
10. [States & feedback](#10-states--feedback)
11. [Content & voice](#11-content--voice)
12. [Representative component code](#12-representative-component-code)

---

## 1. Design principles

InterviewOS is a **learning operating system**, not a dashboard. The product's north
star — *"the user should never have to ask what to study today"* — drives every
visual and interaction decision. Five principles govern the system.

### P1 — Clarity over cleverness
Every screen answers one question unambiguously. The dashboard answers *"how ready am
I?"*; the Today view answers *"what do I do now?"*. Information hierarchy is ruthless:
one primary action per view, secondary actions demoted, tertiary actions hidden behind
menus. Numbers carry units. Charts carry a one-line takeaway, never a naked plot.

### P2 — Focus: "what do I do today"
The Today view is the heart of the app. It is reachable in one keystroke (`g t`),
it is the default landing after intake, and it is designed so a user can act without
reading. Task items are scannable in a vertical glance: icon → title → meta → action.
We never bury today's plan under analytics. Analytics *motivate*; the plan *directs*.

### P3 — Calm density
Senior engineers want information density without noise (think Linear, not Jira). We
achieve density through tight, consistent spacing, restrained color (color = meaning,
not decoration), and generous use of muted text for metadata. Surfaces are layered
subtly with elevation, never with heavy borders or drop-shadows. White space is a
feature; we resist the urge to fill it.

### P4 — Motion with purpose
Animation communicates causality and state change — never decoration for its own sake.
A completed task animates *because* something happened. Page transitions orient the
user spatially. Stagger reveals hierarchy. Every animation is fast (≤ 240 ms for UI,
≤ 400 ms for celebratory moments) and fully respects `prefers-reduced-motion`.

### P5 — Dark-mode-first
Dark mode is the primary, default, designed-first theme — engineers live in dark IDEs.
Light mode is a fully-supported first-class citizen derived from the same semantic
token set, never an afterthought. Both themes meet WCAG AA. We design components in
dark, then verify light.

**Cross-cutting tenets:** keyboard-driven (everything reachable without a mouse),
fast (optimistic UI, skeletons, no layout shift), responsive (mobile through ultrawide),
and accessible (AA is the floor, not the goal).

---

## 2. Design tokens

Tokens are the contract between design and code. We use the **shadcn/ui convention**:
HSL CSS custom properties on `:root` (light) and `.dark` (dark), consumed by Tailwind
via `hsl(var(--token))`. Dark is primary; `.dark` is applied to `<html>` by default
and toggled by `next-themes`.

### 2.1 Color philosophy

- **Neutrals** are a true-neutral slate ramp with a faint cool (blue) cast — calm, not
  clinical. Dark backgrounds sit around `222 18% 7–11%`.
- **Primary** is a confident indigo/violet (Linear-esque) — used for primary actions,
  active nav, focus, and brand moments.
- **Semantic colors** (success/warning/danger/info) are reserved strictly for meaning.
- **Pillar accents** are a distinct, memorable hue per pillar, used in nav, readiness
  cards, radar series, and chart legends — consistent everywhere a pillar appears.

### 2.2 Semantic color tokens — CSS variables (shadcn convention)

All values are **HSL channels** (no `hsl()` wrapper) so Tailwind can compose opacity:
`hsl(var(--primary) / 0.5)`.

```css
/* globals.css */

:root {
  /* ---- Light theme (derived) ---- */
  --background: 0 0% 100%;            /* #FFFFFF page base */
  --foreground: 222 24% 11%;          /* #16181F primary text */

  --surface: 220 20% 98%;             /* #F7F8FA subtle raised area */
  --surface-foreground: 222 24% 11%;

  --card: 0 0% 100%;                  /* #FFFFFF card base */
  --card-foreground: 222 24% 11%;

  --popover: 0 0% 100%;
  --popover-foreground: 222 24% 11%;

  --muted: 220 16% 95%;               /* #EEF0F4 muted fill */
  --muted-foreground: 220 9% 43%;     /* #646B7A muted text (AA on bg) */

  --border: 220 14% 90%;              /* #E1E4EA hairline */
  --input: 220 14% 90%;
  --ring: 248 84% 62%;                /* focus ring = primary */

  --primary: 248 84% 60%;             /* #4F46E5 indigo */
  --primary-foreground: 0 0% 100%;

  --secondary: 220 16% 94%;           /* #EBEEF3 */
  --secondary-foreground: 222 24% 16%;

  --accent: 256 84% 64%;              /* #7C5CFC violet hover/active accent */
  --accent-foreground: 0 0% 100%;

  --success: 152 62% 38%;             /* #25A06A */
  --success-foreground: 0 0% 100%;
  --warning: 36 92% 48%;              /* #EB9A09 */
  --warning-foreground: 30 40% 12%;
  --danger: 0 72% 51%;                /* #DC2626 */
  --danger-foreground: 0 0% 100%;
  --info: 205 90% 48%;                /* #0E97E8 */
  --info-foreground: 0 0% 100%;

  /* charts (generic series, mapped to pillars below) */
  --chart-1: 248 84% 60%;
  --chart-2: 173 70% 40%;
  --chart-3: 25 90% 55%;
  --chart-4: 322 75% 55%;
  --chart-5: 205 90% 48%;
  --chart-grid: 220 14% 90%;

  --radius: 0.625rem;                 /* 10px base radius */
}

.dark {
  /* ---- Dark theme (PRIMARY) ---- */
  --background: 222 20% 7%;           /* #0E1015 app shell base */
  --foreground: 220 18% 96%;          /* #F1F3F7 primary text */

  --surface: 222 18% 10%;             /* #14161C sidebar / topbar */
  --surface-foreground: 220 18% 96%;

  --card: 222 17% 12%;                /* #181B22 card base */
  --card-foreground: 220 18% 96%;

  --popover: 222 18% 11%;             /* #15171E */
  --popover-foreground: 220 18% 96%;

  --muted: 222 14% 18%;               /* #262A33 muted fill */
  --muted-foreground: 220 12% 64%;    /* #969CAA muted text (AA on card) */

  --border: 222 12% 20%;              /* #2C313B hairline */
  --input: 222 12% 22%;
  --ring: 248 90% 68%;                /* brighter ring on dark */

  --primary: 248 90% 66%;             /* #6366F1 indigo (brightened) */
  --primary-foreground: 222 40% 8%;

  --secondary: 222 14% 18%;           /* #262A33 */
  --secondary-foreground: 220 18% 96%;

  --accent: 256 90% 72%;              /* #9B82FF */
  --accent-foreground: 222 40% 8%;

  --success: 152 58% 50%;             /* #34C98A */
  --success-foreground: 222 40% 8%;
  --warning: 38 95% 58%;              /* #F6B12B */
  --warning-foreground: 36 60% 8%;
  --danger: 0 78% 63%;                /* #F0584F */
  --danger-foreground: 222 40% 8%;
  --info: 205 92% 60%;                /* #36AEF5 */
  --info-foreground: 222 40% 8%;

  --chart-1: 248 90% 66%;
  --chart-2: 173 72% 52%;
  --chart-3: 25 95% 62%;
  --chart-4: 322 80% 66%;
  --chart-5: 205 92% 60%;
  --chart-grid: 222 12% 20%;
}
```

### 2.3 Pillar accent colors

Six pillars, six fixed hues. Each ships as a `--pillar-*` HSL channel set and a
foreground. Used in nav indicators, readiness card accents, radar series, badges, and
heatmap legends. Order and meaning are stable across the app.

| Pillar | Token | Dark hex | Light hex | Icon (lucide) |
|--------|-------|----------|-----------|---------------|
| DSA | `--pillar-dsa` | `#6366F1` indigo | `#4F46E5` | `binary` |
| System Design | `--pillar-system-design` | `#22D3A8` teal | `#0EA47F` | `network` |
| LLD | `--pillar-lld` | `#F6B12B` amber | `#D98309` | `boxes` |
| Backend Eng | `--pillar-backend` | `#36AEF5` sky | `#0E97E8` | `server` |
| Behavioral | `--pillar-behavioral` | `#F472B6` pink | `#DB2777` | `messages-square` |
| Resume | `--pillar-resume` | `#A78BFA` violet | `#7C5CFC` | `file-text` |

```css
.dark {
  --pillar-dsa: 248 90% 66%;
  --pillar-system-design: 165 72% 48%;
  --pillar-lld: 38 95% 58%;
  --pillar-backend: 205 92% 60%;
  --pillar-behavioral: 330 88% 70%;
  --pillar-resume: 255 92% 76%;
  --pillar-foreground: 222 40% 8%;
}
:root {
  --pillar-dsa: 248 84% 60%;
  --pillar-system-design: 165 84% 35%;
  --pillar-lld: 33 92% 44%;
  --pillar-backend: 205 90% 48%;
  --pillar-behavioral: 330 81% 50%;
  --pillar-resume: 256 84% 64%;
  --pillar-foreground: 0 0% 100%;
}
```

### 2.4 Typography

**Primary UI font:** `Geist` (via `next/font`), with `Inter` as fallback — both are
neutral, screen-optimized grotesques. **Monospace:** `Geist Mono` / `JetBrains Mono`
for code, problem IDs, time-complexity, and `<kbd>`.

```css
:root {
  --font-sans: "Geist", "Inter", ui-sans-serif, system-ui, -apple-system, sans-serif;
  --font-mono: "Geist Mono", "JetBrains Mono", ui-monospace, "SF Mono", monospace;
}
```

**Type scale** (1.250 major-third-ish, tuned for density). Sizes in `rem` (root 16px):

| Token | px | rem | line-height | weight | Use |
|-------|----|-----|-------------|--------|-----|
| `text-display` | 40 | 2.5 | 1.1 | 700 | Marketing / readiness hero number |
| `text-h1` | 30 | 1.875 | 1.2 | 700 | Page title |
| `text-h2` | 24 | 1.5 | 1.25 | 600 | Section title |
| `text-h3` | 20 | 1.25 | 1.3 | 600 | Card title / subsection |
| `text-lg` | 18 | 1.125 | 1.4 | 500 | Lead / emphasized body |
| `text-base` | 16 | 1.0 | 1.5 | 400 | Body default |
| `text-sm` | 14 | 0.875 | 1.45 | 400 | Secondary body, table cells, task meta |
| `text-xs` | 12 | 0.75 | 1.4 | 500 | Labels, badges, captions, kbd |
| `text-2xs` | 11 | 0.6875 | 1.3 | 600 | Micro-labels (uppercase eyebrows) |

**Weights:** 400 regular, 500 medium (default for labels/buttons), 600 semibold
(headings), 700 bold (display/hero numbers only). Never use 300/100.
**Tracking:** `-0.011em` on `text-h1`/`h2`/`display`; `0.04em` uppercase on `text-2xs`
eyebrows; default elsewhere. **Tabular numerals** (`font-feature-settings: "tnum"`) on
all metrics, timers, and table numbers to prevent jitter.

### 2.5 Spacing scale

4px base grid. Tailwind's default scale extended with two micro steps.

| Token | px | Use |
|-------|----|-----|
| `0.5` | 2 | hairline gaps, icon nudges |
| `1` | 4 | inline gap |
| `1.5` | 6 | tight padding |
| `2` | 8 | default inner gap |
| `3` | 12 | control padding-x, list-item gap |
| `4` | 16 | card padding (compact), stack gap |
| `5` | 20 | card padding (default) |
| `6` | 24 | section gap |
| `8` | 32 | large section gap |
| `10` | 40 | page top padding |
| `12` | 48 | hero spacing |
| `16` | 64 | empty-state vertical rhythm |

**Layout constants:** sidebar `260px` (collapsed `64px`), top bar `56px`, content
max-width `1200px` (dashboard) / `768px` (editor/reading), gutter `24px` desktop /
`16px` mobile.

### 2.6 Border radius

```js
borderRadius: {
  sm: "calc(var(--radius) - 4px)",  // 6px  — inputs, badges, kbd
  md: "calc(var(--radius) - 2px)",  // 8px  — buttons, small cards
  lg: "var(--radius)",              // 10px — cards, popovers, menus
  xl: "calc(var(--radius) + 6px)",  // 16px — modals, large panels
  "2xl": "calc(var(--radius) + 14px)", // 24px — hero/feature surfaces
  full: "9999px",                   // pills, avatars, progress ring caps
}
```

### 2.7 Shadows / elevation

Dark mode leans on **layered surface lightness + subtle borders** more than shadows;
shadows are soft and low-opacity. Light mode uses crisper ambient shadows.

| Token | Use | Dark | Light |
|-------|-----|------|-------|
| `elevation-0` | flush surfaces | none (border only) | none |
| `elevation-1` | cards | `0 1px 2px hsl(222 40% 2% / .4)` | `0 1px 2px hsl(220 20% 20% / .06)` |
| `elevation-2` | dropdowns, popovers | `0 4px 16px hsl(222 40% 2% / .5)` | `0 4px 16px hsl(220 20% 20% / .10)` |
| `elevation-3` | dialogs, command palette | `0 16px 48px hsl(222 40% 1% / .6)` | `0 16px 48px hsl(220 20% 20% / .16)` |
| `glow-primary` | focus / active brand | `0 0 0 1px hsl(var(--primary) / .4), 0 0 24px hsl(var(--primary) / .25)` | same, lower alpha |

### 2.8 Z-index scale

| Token | Value | Layer |
|-------|-------|-------|
| `base` | 0 | page content |
| `sticky` | 10 | sticky headers, table header |
| `nav` | 20 | sidebar, top bar |
| `dropdown` | 30 | menus, popovers, tooltips |
| `overlay` | 40 | dialog/sheet backdrop |
| `modal` | 50 | dialog, sheet, command palette |
| `toast` | 60 | toasts (always on top) |

### 2.9 Breakpoints

| Token | Min width | Target |
|-------|-----------|--------|
| `sm` | 640px | large phone / small tablet |
| `md` | 768px | tablet — sidebar becomes collapsible drawer ≤ this |
| `lg` | 1024px | laptop — full app shell |
| `xl` | 1280px | desktop — default design target |
| `2xl` | 1536px | large desktop / ultrawide (content stays centered) |

### 2.10 Tailwind `theme.extend` config snippet

```ts
// tailwind.config.ts
import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: "class",
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        surface: { DEFAULT: "hsl(var(--surface))", foreground: "hsl(var(--surface-foreground))" },
        card: { DEFAULT: "hsl(var(--card))", foreground: "hsl(var(--card-foreground))" },
        popover: { DEFAULT: "hsl(var(--popover))", foreground: "hsl(var(--popover-foreground))" },
        muted: { DEFAULT: "hsl(var(--muted))", foreground: "hsl(var(--muted-foreground))" },
        primary: { DEFAULT: "hsl(var(--primary))", foreground: "hsl(var(--primary-foreground))" },
        secondary: { DEFAULT: "hsl(var(--secondary))", foreground: "hsl(var(--secondary-foreground))" },
        accent: { DEFAULT: "hsl(var(--accent))", foreground: "hsl(var(--accent-foreground))" },
        success: { DEFAULT: "hsl(var(--success))", foreground: "hsl(var(--success-foreground))" },
        warning: { DEFAULT: "hsl(var(--warning))", foreground: "hsl(var(--warning-foreground))" },
        danger: { DEFAULT: "hsl(var(--danger))", foreground: "hsl(var(--danger-foreground))" },
        info: { DEFAULT: "hsl(var(--info))", foreground: "hsl(var(--info-foreground))" },
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        pillar: {
          dsa: "hsl(var(--pillar-dsa))",
          "system-design": "hsl(var(--pillar-system-design))",
          lld: "hsl(var(--pillar-lld))",
          backend: "hsl(var(--pillar-backend))",
          behavioral: "hsl(var(--pillar-behavioral))",
          resume: "hsl(var(--pillar-resume))",
          foreground: "hsl(var(--pillar-foreground))",
        },
        chart: {
          1: "hsl(var(--chart-1))", 2: "hsl(var(--chart-2))", 3: "hsl(var(--chart-3))",
          4: "hsl(var(--chart-4))", 5: "hsl(var(--chart-5))", grid: "hsl(var(--chart-grid))",
        },
      },
      fontFamily: { sans: ["var(--font-sans)"], mono: ["var(--font-mono)"] },
      fontSize: {
        "2xs": ["0.6875rem", { lineHeight: "1.3", fontWeight: "600", letterSpacing: "0.04em" }],
        xs: ["0.75rem", { lineHeight: "1.4" }],
        sm: ["0.875rem", { lineHeight: "1.45" }],
        base: ["1rem", { lineHeight: "1.5" }],
        lg: ["1.125rem", { lineHeight: "1.4" }],
        h3: ["1.25rem", { lineHeight: "1.3", fontWeight: "600" }],
        h2: ["1.5rem", { lineHeight: "1.25", fontWeight: "600", letterSpacing: "-0.011em" }],
        h1: ["1.875rem", { lineHeight: "1.2", fontWeight: "700", letterSpacing: "-0.011em" }],
        display: ["2.5rem", { lineHeight: "1.1", fontWeight: "700", letterSpacing: "-0.02em" }],
      },
      borderRadius: {
        sm: "calc(var(--radius) - 4px)", md: "calc(var(--radius) - 2px)",
        lg: "var(--radius)", xl: "calc(var(--radius) + 6px)", "2xl": "calc(var(--radius) + 14px)",
      },
      boxShadow: {
        "elevation-1": "0 1px 2px hsl(222 40% 2% / 0.4)",
        "elevation-2": "0 4px 16px hsl(222 40% 2% / 0.5)",
        "elevation-3": "0 16px 48px hsl(222 40% 1% / 0.6)",
        "glow-primary": "0 0 0 1px hsl(var(--primary) / 0.4), 0 0 24px hsl(var(--primary) / 0.25)",
      },
      zIndex: { sticky: "10", nav: "20", dropdown: "30", overlay: "40", modal: "50", toast: "60" },
      keyframes: {
        "fade-in": { from: { opacity: "0" }, to: { opacity: "1" } },
        "slide-up": { from: { opacity: "0", transform: "translateY(8px)" }, to: { opacity: "1", transform: "translateY(0)" } },
        "scale-in": { from: { opacity: "0", transform: "scale(0.96)" }, to: { opacity: "1", transform: "scale(1)" } },
        shimmer: { "100%": { transform: "translateX(100%)" } },
      },
      animation: {
        "fade-in": "fade-in 160ms ease-out",
        "slide-up": "slide-up 200ms cubic-bezier(0.22, 1, 0.36, 1)",
        "scale-in": "scale-in 160ms cubic-bezier(0.22, 1, 0.36, 1)",
        shimmer: "shimmer 1.6s infinite",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
};
export default config;
```

---

## 3. Component inventory (primitives)

All primitives are built on **shadcn/ui** (Radix under the hood). Conventions: every
interactive element uses the `focus-visible:ring-2 ring-ring ring-offset-2
ring-offset-background` focus pattern; disabled = `opacity-50
pointer-events-none`; all variants generated via `cva` (class-variance-authority).

| Component | Purpose | Key variants / states | A11y notes |
|-----------|---------|------------------------|------------|
| **Button** | Primary action affordance | `variant`: primary, secondary, outline, ghost, destructive, link · `size`: sm(28h), default(36h), lg(40h), icon · states: default/hover/active/focus/disabled/loading (spinner replaces label, keeps width) | Real `<button>`; `aria-disabled` while loading; `aria-label` required for icon-only |
| **Input** | Single-line text | states: default, focus, error (`aria-invalid`), disabled · with leading/trailing icon or addon | Labeled via `<Label htmlFor>`; error text via `aria-describedby` |
| **Textarea** | Multi-line (notes, STAR) | auto-grow, char counter, error | Same labeling; `aria-describedby` for counter/error |
| **Select** | Single choice, short list | open/closed, selected, disabled item, grouped | Radix Select: full keyboard, typeahead, `aria-activedescendant` |
| **Combobox** | Searchable single/multi select (companies, topics) | search, async loading, multi-tag, empty, create-new | `role=combobox`, `aria-expanded`, listbox semantics |
| **Checkbox** | Boolean / multi-select | unchecked, checked, indeterminate, disabled, error | `role=checkbox`, space toggles, label clickable |
| **Radio group** | One-of-many | selected, focus roving, disabled | Roving tabindex, arrow-key nav |
| **Switch** | Instant on/off (settings, theme) | on, off, disabled | `role=switch` `aria-checked`; not for forms needing submit |
| **Tabs** | Sectioned content (topic detail) | active, hover, disabled · underline & pill styles | Radix Tabs: arrow nav, `aria-selected`, panel `aria-labelledby` |
| **Dialog / Modal** | Focused interruption (confirm, edit) | sm/default/lg widths, scrollable body | Focus trap, `Esc` close, restore focus, `aria-modal`, labelled title |
| **Sheet / Drawer** | Side/bottom panel (topic detail, filters) | side: right(default)/left/bottom · sizes | Same trap/restore; bottom variant on mobile |
| **Dropdown menu** | Contextual actions (task ⋯, profile) | items, checkbox/radio items, submenus, separators, shortcuts | Radix Menu: full keyboard, typeahead, `role=menu` |
| **Command palette (⌘K)** | Global action/nav search | empty, results grouped, loading, recent, nested pages | `cmdk` lib; `role=dialog`+listbox; arrow/enter; `Esc` close |
| **Toast** | Transient confirmation/error | success, error, info, warning, with action/undo · auto-dismiss 4s (errors 6s, sticky on hover) | `role=status` (polite) / `role=alert` (errors); `sonner` |
| **Tooltip** | Supplemental label on hover/focus | default; delay 400ms open, 0 close | Keyboard-focus triggers; never sole carrier of critical info |
| **Popover** | Lightweight floating panel (confidence picker, date) | default; arrow optional | `Esc` close, focus trap optional, click-outside dismiss |
| **Card** | Content container | default, interactive (hover lift), bordered, accent (pillar) · slots: header/content/footer | Region/article semantics where meaningful |
| **Badge** | Status/metadata token | variant: default, secondary, outline, success, warning, danger, info, pillar-* · size sm/default | Color never sole signal — pair with text/icon |
| **Progress bar** | Linear completion | determinate, indeterminate · sized 4/6/8 px · pillar-tinted | `role=progressbar` with `aria-valuenow/min/max` |
| **Progress ring** | Compact % (readiness) | sizes 40/56/72/96 · animated sweep · center label | `role=img` `aria-label="68% ready"` |
| **Avatar** | User identity | image, initials fallback, sizes xs–lg, status dot | `alt` text; initials when no image |
| **Table (TanStack)** | Dense tabular data (problems) | sortable headers, row hover/selected, sticky header, pagination, column visibility, row actions | `<table>` semantics, `aria-sort`, keyboard row nav |
| **Calendar** | Date picker (intake, reschedule) | range, single, disabled days, today marker | `react-day-picker`; grid keyboard nav, `aria-label` per day |
| **Heatmap** | Activity calendar (streak) | 5 intensity buckets, hover tooltip, month labels | `<svg>` with per-cell `<title>` + `aria-label` |
| **Charts (Recharts)** | line/area/radar/bar | themed via chart tokens, tooltip, legend, responsive container | Provide accessible summary + data table fallback (`aria-describedby`) |
| **Skeleton** | Loading placeholder | shimmer, shapes match real content | `aria-hidden`; live region announces "Loading" |
| **Empty state** | No-data guidance | illustration/icon + title + body + CTA | Heading + descriptive action |
| **Kbd** | Keyboard hint | single key, chord (`g` then `d`), combo (`⌘K`) | `<kbd>`; decorative, mirrors real binding |

### 3.1 Button anatomy (cva reference)

```ts
const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium " +
  "transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring " +
  "focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none " +
  "disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        primary: "bg-primary text-primary-foreground hover:bg-primary/90 active:bg-primary/80",
        secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        outline: "border border-border bg-transparent hover:bg-muted",
        ghost: "hover:bg-muted hover:text-foreground",
        destructive: "bg-danger text-danger-foreground hover:bg-danger/90",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        sm: "h-7 px-2.5 text-xs", default: "h-9 px-4", lg: "h-10 px-6 text-base",
        icon: "size-9 p-0",
      },
    },
    defaultVariants: { variant: "primary", size: "default" },
  }
);
```

---

## 4. App-specific components

These compose primitives into the domain. Each is a named, reusable component with a
typed prop contract.

### 4.1 ReadinessCard
Surfaces overall or per-pillar readiness on the dashboard.
- **Slots:** pillar icon + name (or "Overall"), `ProgressRing` (the % hero), delta vs.
  last week (`↑ 4%` in success / `↓` in danger), one-line takeaway ("System Design is
  your weakest pillar").
- **Variants:** `overall` (large, primary ring) · `pillar` (compact, pillar-tinted ring).
- **States:** loading (skeleton ring + lines), no-data ("Not started").
- **A11y:** ring `aria-label`; delta announced with direction word, not just arrow.

### 4.2 PillarRadarChart
Recharts `RadarChart` plotting the six pillars' readiness on one polygon.
- Series colored with `--primary` fill at low opacity; axis labels are pillar names
  with their pillar dots. Optional second series = "target readiness" outline.
- Tooltip shows pillar, value %, and rank. Responsive; collapses axis labels to icons < `sm`.
- **A11y:** `aria-label` summary + visually-hidden data table.

### 4.3 StreakHeatmap
GitHub-style calendar of `StudyDay` activity (last 26 weeks, weeks as columns).
- 5 intensity buckets driven by minutes studied; empty cell = `muted`, max =
  `success`. Hover tooltip: "2h 15m · 4 tasks · Jun 12". Month labels on top,
  weekday labels (M/W/F) on left.
- **States:** loading (dimmed grid), empty ("Your streak starts today").
- **A11y:** each cell `<rect>` has `aria-label="Jun 12: 2h 15m studied"`; summary heading.

### 4.4 TodayTaskList & TodayTaskItem
The product's centerpiece (see code in §12).
- **TodayTaskList:** grouped by pillar or flat; sticky day header with date + remaining
  count + "X of Y done" progress bar; supports reorder-free read; empty/all-done states.
- **TodayTaskItem** anatomy: `[complete checkbox] [kind icon] title + meta line
  (pillar badge · est. minutes · difficulty · "Revise" tag) [confidence picker]
  [⋯ menu]`.
  - **Kind icons** (each in its kind tint): study `book-open`, solve `code`, read
    `file-text`, watch `play`, revise `rotate-ccw`, mock `mic`.
  - **Confidence picker:** 1–5 popover (or inline dots), filled with pillar accent;
    appears on complete or via `e`.
  - **States:** todo, completing (optimistic, struck-through + checkmark draw), done
    (muted, collapsed meta), overdue/rescheduled (warning tag), skipped (dimmed).

### 4.5 RoadmapTimeline (Week / Day)
Vertical timeline of `RoadmapWeek` → `PlanDay`.
- Week header: "Week 3 · Graphs & Trees" with week progress bar and date range.
- Day node: dot on a spine (filled = complete, ring = today, hollow = future),
  expandable to that day's task list. Today is auto-scrolled into view and ringed.
- **A11y:** ordered list semantics; today marked `aria-current="date"`.

### 4.6 TopicDetailPanel
Opens as a `Sheet` (right) or full page. Tabs: Concept · Resources · Problems ·
Revision · Notes. Header carries pillar badge, confidence ring, time-spent, and a
"Mark studied" primary action. Resources render as `ResourceRow`s with type icon +
est. time + priority.

### 4.7 ProblemRow
Dense row for the DSA problem table (TanStack).
- Columns: status dot, title (links out to LeetCode, `external-link` icon), pattern
  badge(s), difficulty pill (Easy=success / Medium=warning / Hard=danger), company
  frequency sparkline/count, confidence dots, last-revised relative time, ⋯ actions.
- Hover reveals quick actions; row selectable for bulk reschedule.

### 4.8 StarStoryEditor (Behavioral)
Four labeled sections — **S**ituation / **T**ask / **A**ction / **R**esult — each an
auto-grow `Textarea` with a soft word-count guide and inline "Improve with AI" ghost
button. Theme tag selector (leadership/conflict/…), metrics chips, and a live "Story
strength" meter (length/metrics/specificity heuristic, AI-augmented). Autosaves with a
"Saved" toast-free inline indicator.

### 4.9 ResumeScorer
Score header = large `ProgressRing` (ATS score 0–100, color-graded). Below: categorized
checklist (Impact, Metrics, Keywords, Formatting) with pass/warn/fail badges and
expandable fixes. Keyword-match chips show matched (success) vs. missing (muted-outline)
keywords against the target role.

### 4.10 MockResultCard
Summarizes a `MockInterview`. Header: type icon (coding/SD/behavioral), date, overall
band (Strong Hire → No Hire) as a graded badge. Body: per-dimension bar ratings,
strengths/weaknesses lists, and auto-generated remediation tasks ("3 follow-ups added
to your plan") that deep-link into Today.

### 4.11 Command palette actions
`cmdk`-driven `⌘K` registry. Grouped: **Navigate** (Today, Dashboard, Roadmap, each
pillar), **Create** (new note, new STAR story), **Actions** (complete next task, start
mock, reschedule day, log study session), **Search** (topics/problems/resources, async),
**Settings** (toggle theme, keyboard help). Each action carries an icon, a `kbd` hint
where one exists, and is fuzzy-searchable. Recents pinned to top on open.

---

## 5. Layout system

### 5.1 App shell

```
┌──────────────────────────────────────────────────────────────────────────┐
│  TOP BAR (h-56, surface, sticky, z-nav)                                    │
│  [≡ collapse] InterviewOS    [🔍 Search / ⌘K ]    🔥 12d  [theme] [avatar▾]│
├───────────────┬────────────────────────────────────────────────────────────┤
│ SIDEBAR       │  MAIN CONTENT (background, scroll)                          │
│ (w-260,       │  ┌──────────────────────────────────────────────────────┐ │
│  surface,     │  │ Page header: title · breadcrumbs · primary action     │ │
│  z-nav)       │  ├──────────────────────────────────────────────────────┤ │
│               │  │ Content (max-w-1200, centered, px-gutter)              │ │
│ ◆ Today       │  │                                                        │ │
│ ▦ Dashboard   │  │                                                        │ │
│ ▤ Roadmap     │  │                                                        │ │
│ ── Pillars ── │  │                                                        │ │
│ ◼ DSA         │  │                                                        │ │
│ ◼ System Des. │  │                                                        │ │
│ ◼ LLD         │  │                                                        │ │
│ ◼ Backend     │  │                                                        │ │
│ ◼ Behavioral  │  │                                                        │ │
│ ◼ Resume      │  │                                                        │ │
│ ── ───────── ─│  │                                                        │ │
│ ◷ Revision    │  │                                                        │ │
│ ◎ Mocks       │  │                                                        │ │
│ ⚙ Settings    │  └──────────────────────────────────────────────────────┘ │
└───────────────┴────────────────────────────────────────────────────────────┘
```

- **Sidebar:** primary nav (Today, Dashboard, Roadmap) on top; **Pillars** group with
  each pillar's accent dot + icon and a tiny readiness mini-bar; utility group
  (Revision, Mocks, Settings) at bottom. Active item: pillar/primary tint left border
  (3px) + `bg-muted` + foreground text. Collapsible to 64px (icons only, tooltips on hover).
- **Top bar:** collapse toggle, wordmark, centered search button (opens `⌘K`), streak
  flame with count, theme toggle, profile dropdown. Stays sticky.

### 5.2 Responsive behavior

| Breakpoint | Shell behavior |
|------------|----------------|
| `< md` (mobile) | Sidebar hidden; hamburger opens it as a left `Sheet`. Bottom-fixed mini nav (Today / Dashboard / Roadmap / ⌘K). Cards single-column. Tables → stacked card rows. |
| `md`–`lg` | Sidebar collapsed (icons) by default; expands on toggle/hover. Two-column grids. |
| `≥ lg` | Full expanded sidebar. Dashboard 3-col grid. Detail opens as right `Sheet` or split. |
| `≥ 2xl` | Content max-width holds at 1200/1280px and centers; no full-bleed stretch. |

### 5.3 Grid system
12-column CSS grid, `gap-6`, gutters `px-6` (`px-4` mobile). Dashboard uses named
areas: readiness hero spans 8 cols / radar 4; Today list spans 8 / streak+upcoming 4;
charts row 6/6. Use `grid-cols-1 md:grid-cols-2 xl:grid-cols-3` utility patterns for
card grids.

### 5.4 Page templates

| Template | Layout | Examples |
|----------|--------|----------|
| **Dashboard** | Header + 12-col card grid; readiness hero, radar, Today preview, streak, charts | Home/Dashboard |
| **List** | Header + filter/toolbar row + TanStack table or card grid + pagination | Problems, Resources, Mocks |
| **Detail** | Sticky sub-header + tabbed content (max-w-768 reading width) or right Sheet | Topic, Design Problem |
| **Editor** | Narrow centered column (max-w-768), autosave indicator, sticky action bar | STAR story, Notes, Resume |
| **Wizard / Intake** | Centered card (max-w-560), step indicator, back/next, progress | Intake, onboarding |

---

## 6. Keyboard shortcuts & interaction model

InterviewOS is keyboard-driven. The model: **`⌘K` is the universal entry point**;
**single keys** act on the focused context; **`g`-prefixed chords** navigate
("go to"). Shortcuts are discoverable via `?` (shortcut cheat-sheet dialog) and shown
inline in menus/tooltips. Bindings are suppressed while typing in inputs/textareas
(except `⌘K`, `Esc`).

| Shortcut | Action | Scope |
|----------|--------|-------|
| `⌘K` / `Ctrl K` | Open command palette | Global |
| `/` | Focus search / open palette in search mode | Global |
| `?` | Open keyboard shortcuts help | Global |
| `g` then `t` | Go to Today | Global |
| `g` then `d` | Go to Dashboard | Global |
| `g` then `r` | Go to Roadmap | Global |
| `g` then `p` | Go to Problems (DSA) | Global |
| `g` then `b` | Go to Behavioral | Global |
| `g` then `v` | Go to Revision (due) | Global |
| `g` then `s` | Go to Settings | Global |
| `j` / `k` | Move focus down / up in lists | List/Today |
| `c` | Complete focused task | Today/task |
| `e` | Edit focused task (confidence/notes) | Today/task |
| `r` | Reschedule focused task/day | Today/Roadmap |
| `x` | Toggle select (bulk) | List |
| `1`–`5` | Set confidence 1–5 (when picker focused) | Task |
| `n` | New note / new story (context-aware) | Editor/list |
| `⌘ Enter` | Save & close (forms/editors) | Editor/dialog |
| `Esc` | Close dialog/sheet/popover/palette; clear selection | Global |
| `[` / `]` | Collapse / expand sidebar | Global |
| `⌘ \` | Toggle theme | Global |
| `Tab` / `Shift Tab` | Standard focus traversal | Global |
| `←/→/↑/↓` | Radix component nav (menus, tabs, calendar, radar legend) | Component |

**Interaction rules:** chords have a 1s timeout; `Esc` always backs out one layer;
focus is never trapped except in modals/sheets; on route change focus moves to the page
`<h1>` (announced to SR). All shortcuts are no-ops (not errors) when not applicable.

---

## 7. Motion & animation

Powered by **Framer Motion**. Motion encodes meaning; it is fast, subtle, and
interruptible.

### 7.1 Duration & easing tokens

| Token | Duration | Easing | Use |
|-------|----------|--------|-----|
| `motion.instant` | 80ms | `ease-out` | hover/press feedback, color fades |
| `motion.fast` | 160ms | `ease-out` | tooltips, dropdown, fade-in |
| `motion.base` | 200ms | `cubic-bezier(0.22,1,0.36,1)` (emphasized-decel) | dialogs, sheets, slide-up, list items |
| `motion.slow` | 320ms | `cubic-bezier(0.22,1,0.36,1)` | page transitions, sheet from edge |
| `motion.celebrate` | 400ms | spring `{ stiffness: 380, damping: 22 }` | task completion, streak milestone |

```ts
export const EASE = { out: [0, 0, 0.2, 1], emphasized: [0.22, 1, 0.36, 1] } as const;
export const DUR = { instant: 0.08, fast: 0.16, base: 0.2, slow: 0.32 } as const;
```

### 7.2 Patterns

- **Page transition:** content fades + 8px slide-up (`motion.base`); shared layout via
  `AnimatePresence mode="wait"`. Sidebar/topbar persist (no re-animate).
- **List stagger:** Today tasks / cards enter with `staggerChildren: 0.04`, each child
  fade + slide-up 8px. Cap stagger at 12 items, then batch.
- **Dialog / sheet:** backdrop fade (`motion.fast`); dialog `scale 0.96→1` + fade
  (`motion.base`); sheet slides from edge (`motion.slow`).
- **Task completion (celebrate):** checkbox draws a check (SVG path), row lightly
  scales (`1→1.02→1`) with a spring, title strikes through, a subtle pillar-tinted
  glow pulses once, optional confetti burst on the *last* task of the day / streak
  milestone only (never per-task spam).
- **Progress ring / bar:** animate `value` on mount and on change with `motion.base`
  (ease the sweep; never instant jump).
- **Hover/press:** interactive cards lift 1px + `elevation-2`; buttons darken via color
  transition only (no scale) for snappiness.
- **Skeleton shimmer:** the `shimmer` keyframe (§2.10), 1.6s loop.
- **Number count-up:** readiness/streak numbers count from previous to new value over
  `motion.base` using `animate()` on a motion value.

### 7.3 Reduced motion
Respect `prefers-reduced-motion: reduce` globally. Implementation: a `useReducedMotion()`
gate that swaps all transforms/sprites for **opacity-only** crossfades (≤120ms) and
disables stagger, confetti, count-up (snap to value), and the completion scale/glow.
Provide a global CSS guard:

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }
}
```

---

## 8. Accessibility (WCAG 2.1 AA)

AA is the floor. Verified per release.

- **Contrast:** body text ≥ 4.5:1, large text (≥ 24px or 19px bold) and UI/icon
  boundaries ≥ 3:1. All semantic tokens above are tuned to pass on their intended
  surfaces (e.g. `muted-foreground` on `card`/`background` in both themes). Pillar and
  status colors are **never the sole carrier of meaning** — always paired with text or icon.
- **Focus:** visible `focus-visible` ring (`ring-2 ring-ring ring-offset-2`) on every
  interactive element; ring color is the brightened `--ring` for ≥ 3:1 against
  surfaces. Never remove outlines without a replacement.
- **Keyboard:** everything operable without a mouse (see §6). Logical tab order; no
  traps outside modals; `Esc` exits. Roving tabindex on composite widgets (handled by
  Radix). Skip-to-content link as first focusable element.
- **ARIA & semantics:** prefer native elements; add ARIA only to fill gaps. Landmarks:
  `<header>`, `<nav aria-label="Primary">`, `<main>`, `<aside>`. Buttons are buttons;
  links are links. Icon-only controls require `aria-label`. Live regions: `role=status`
  (polite) for toasts/autosave, `role=alert` for errors, and a polite region announcing
  loading and optimistic completions ("Task completed").
- **Charts:** every Recharts/heatmap/radar carries a text summary and a
  visually-hidden data table (`aria-describedby`) so the data is reachable by SR users.
- **Forms:** every field has a visible `<Label htmlFor>`; errors via `aria-invalid` +
  `aria-describedby`; required state communicated in text, not color/asterisk alone.
- **Reduced motion:** honored globally (§7.3).
- **Targets:** interactive hit area ≥ 24×24px (≥ 44px primary touch targets on mobile).
- **Theming:** both themes meet AA; `next-themes` respects system preference; theme is
  persisted and announced on toggle.

---

## 9. Iconography & illustration

- **Library:** **lucide-react** exclusively (consistent 1.5px stroke, 24px grid). No
  mixing icon sets. Default render size `16px` (`size-4`) inline / `20px` in nav.
- **Rules:** icons reinforce labels, rarely replace them; icon-only buttons get
  `aria-label` + tooltip. Stroke width stays at the lucide default; never recolor
  arbitrarily — icons inherit `currentColor` or a token (pillar/status).
- **Pillar icons (fixed):** DSA `binary`, System Design `network`, LLD `boxes`, Backend
  `server`, Behavioral `messages-square`, Resume `file-text`.
- **Task-kind icons (fixed):** study `book-open`, solve `code`, read `file-text`, watch
  `play`, revise `rotate-ccw`, mock `mic`.
- **Status/utility:** streak `flame`, readiness `gauge`, due/overdue `clock`, success
  `check`/`check-circle-2`, warning `triangle-alert`, danger `octagon-alert`, info
  `info`, external `external-link`, search `search`, command `command`.
- **Illustration:** spare, geometric line-art in muted tones with a single pillar/primary
  accent; reserved for empty states and onboarding. No stock 3D mascots or gradients that
  fight the calm aesthetic. SVG, theme-aware via `currentColor`.

---

## 10. States & feedback

Every data-bound surface defines four states: **loading, empty, error, success/loaded.**

- **Loading (skeleton):** show content-shaped skeletons (`Skeleton` shimmer) that match
  final layout to avoid layout shift — never a centered spinner for primary content.
  Spinners only inside buttons or for tiny inline async. Charts show a dimmed axis frame.
- **Empty:** purposeful empty states with icon/illustration + heading + one-line body +
  a single CTA. Tone is coach-like: "Your streak starts today — complete your first
  task." Distinguish *empty-by-newness* (encouraging CTA) from *empty-by-filter*
  ("No problems match these filters — clear filters").
- **Error:** inline, scoped, recoverable. Field errors below the field; section errors
  in a bordered `danger` callout with a Retry. Network/global errors via `alert` toast
  with Retry. Never blank the screen; preserve prior good data where possible. Error
  copy says what happened + what to do, never a stack trace.
- **Success / confirmation:** prefer in-context feedback (checkmark draw, count-up,
  inline "Saved") over toasts; reserve toasts for actions whose result isn't visible
  on screen, always with **Undo** for destructive/consequential actions.

**Optimistic UI (task completion — the canonical flow):**
1. User presses `c` / clicks checkbox.
2. UI updates instantly: checkbox animates, title strikes through, item moves to done,
   day progress bar + readiness ring animate, "Task completed" announced (polite).
3. Mutation fires (TanStack Query `useMutation` with `onMutate` snapshot).
4. On error: rollback to the snapshot, re-insert the item, show `danger` toast with
   Retry. On success: silently reconcile (no extra toast).
5. The Revision Engine's scheduled follow-up appears as a subtle "Revision scheduled +1d"
   inline note, not a modal.

---

## 11. Content & voice

The product is a **calm, competent coach** — encouraging without being saccharine,
concise without being terse. It respects the user's intelligence and time.

- **Tone:** direct, supportive, second person ("You're 68% ready"). Celebrate progress,
  normalize gaps ("System Design is your next focus — that's expected at this stage").
- **Concision:** labels are nouns/short verbs ("Complete", "Reschedule", not "Click here
  to mark this task as complete"). One sentence beats three.
- **Action-first:** buttons describe the outcome ("Generate roadmap", "Mark studied",
  "Improve with AI"), never generic ("Submit", "OK").
- **Numbers with meaning:** always pair a metric with a takeaway or unit ("12-day streak",
  "Est. ready: Aug 14").
- **Errors:** plain, blameless, actionable ("Couldn't save your note. Check your
  connection and retry." — never "Error 500").
- **Empty/onboarding:** forward-looking and motivating ("Your 12-week plan is ready.
  Here's today.").
- **No dark patterns, no nagging, no exclamation spam.** Encouragement is earned and
  specific, not constant.

Microcopy reference:
| Context | Copy |
|---------|------|
| Today, all done | "That's everything for today. Nicely done." |
| Today, empty (rest day) | "No tasks today — rest is part of the plan." |
| Readiness hero | "You're 68% ready · Est. interview-ready Aug 14" |
| Streak milestone | "12-day streak. Keep it going." |
| Task completed | "Done. Revision scheduled in 1 day." |
| Mock result | "Strong on coding, work on system design depth." |

---

## 12. Representative component code

Two anchor components in the project's shadcn/Tailwind/Framer Motion idiom. Assume
`cn()` (clsx + tailwind-merge), shadcn primitives, lucide icons, and TanStack Query.

### 12.1 `ReadinessCard`

```tsx
// components/dashboard/readiness-card.tsx
"use client";

import { motion, useReducedMotion } from "framer-motion";
import { ArrowDownRight, ArrowUpRight, type LucideIcon } from "lucide-react";
import { ProgressRing } from "@/components/ui/progress-ring";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

type PillarKey = "dsa" | "system-design" | "lld" | "backend" | "behavioral" | "resume";

interface ReadinessCardProps {
  label: string;                 // "Overall" | "System Design"
  value: number;                 // 0–100
  deltaPct?: number;             // vs last week, signed
  takeaway?: string;             // one-line insight
  icon: LucideIcon;
  pillar?: PillarKey;            // omit for Overall (uses primary)
  variant?: "overall" | "pillar";
  loading?: boolean;
}

export function ReadinessCard({
  label, value, deltaPct, takeaway, icon: Icon, pillar,
  variant = "pillar", loading = false,
}: ReadinessCardProps) {
  const reduce = useReducedMotion();
  const ringColor = pillar ? `hsl(var(--pillar-${pillar}))` : "hsl(var(--primary))";
  const isUp = (deltaPct ?? 0) >= 0;

  if (loading) {
    return (
      <Card className="flex items-center gap-4 p-5">
        <div className="size-16 shrink-0 animate-pulse rounded-full bg-muted" />
        <div className="flex-1 space-y-2">
          <div className="h-3 w-24 animate-pulse rounded bg-muted" />
          <div className="h-3 w-32 animate-pulse rounded bg-muted" />
        </div>
      </Card>
    );
  }

  return (
    <motion.div
      initial={reduce ? { opacity: 0 } : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: [0.22, 1, 0.36, 1] }}
    >
      <Card
        className={cn(
          "group flex items-center gap-4 p-5 transition-shadow hover:shadow-elevation-2",
          variant === "overall" && "p-6",
        )}
      >
        <ProgressRing
          value={value}
          size={variant === "overall" ? 88 : 64}
          color={ringColor}
          ariaLabel={`${label} readiness ${value} percent`}
        />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <Icon className="size-4 text-muted-foreground" aria-hidden />
            <span className="text-2xs uppercase text-muted-foreground">{label}</span>
          </div>
          <div className="mt-1 flex items-baseline gap-2">
            <span className="text-h2 font-bold tabular-nums">{value}%</span>
            {deltaPct !== undefined && (
              <span
                className={cn(
                  "flex items-center gap-0.5 text-xs font-medium tabular-nums",
                  isUp ? "text-success" : "text-danger",
                )}
              >
                {isUp ? <ArrowUpRight className="size-3.5" /> : <ArrowDownRight className="size-3.5" />}
                {Math.abs(deltaPct)}%
                <span className="sr-only">{isUp ? "up" : "down"} from last week</span>
              </span>
            )}
          </div>
          {takeaway && (
            <p className="mt-1 truncate text-sm text-muted-foreground">{takeaway}</p>
          )}
        </div>
      </Card>
    </motion.div>
  );
}
```

### 12.2 `TodayTaskItem`

```tsx
// components/today/today-task-item.tsx
"use client";

import { motion, useReducedMotion } from "framer-motion";
import {
  BookOpen, Code, FileText, Play, RotateCcw, Mic, MoreHorizontal, type LucideIcon,
} from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { ConfidencePicker } from "@/components/today/confidence-picker";
import {
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type TaskKind = "study" | "solve" | "read" | "watch" | "revise" | "mock";
type PillarKey = "dsa" | "system-design" | "lld" | "backend" | "behavioral" | "resume";

const KIND_ICON: Record<TaskKind, LucideIcon> = {
  study: BookOpen, solve: Code, read: FileText, watch: Play, revise: RotateCcw, mock: Mic,
};

export interface TodayTask {
  id: string;
  title: string;
  kind: TaskKind;
  pillar: PillarKey;
  pillarLabel: string;
  estMinutes: number;
  difficulty?: "Easy" | "Medium" | "Hard";
  isRevision?: boolean;
  done: boolean;
  confidence?: 1 | 2 | 3 | 4 | 5;
}

interface TodayTaskItemProps {
  task: TodayTask;
  focused?: boolean;                       // for j/k keyboard nav
  onToggleComplete: (id: string, next: boolean) => void;  // optimistic upstream
  onSetConfidence: (id: string, c: 1 | 2 | 3 | 4 | 5) => void;
  onEdit: (id: string) => void;
  onReschedule: (id: string) => void;
}

const DIFF_COLOR = {
  Easy: "success", Medium: "warning", Hard: "danger",
} as const;

export function TodayTaskItem({
  task, focused, onToggleComplete, onSetConfidence, onEdit, onReschedule,
}: TodayTaskItemProps) {
  const reduce = useReducedMotion();
  const KindIcon = KIND_ICON[task.kind];

  return (
    <motion.li
      layout={!reduce}
      initial={reduce ? { opacity: 0 } : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      exit={reduce ? { opacity: 0 } : { opacity: 0, x: 8 }}
      transition={{ duration: 0.2, ease: [0.22, 1, 0.36, 1] }}
      data-focused={focused}
      className={cn(
        "group flex items-center gap-3 rounded-lg border border-border bg-card px-3 py-2.5",
        "transition-colors hover:bg-muted/40",
        "data-[focused=true]:ring-2 data-[focused=true]:ring-ring",
        task.done && "opacity-60",
      )}
    >
      <Checkbox
        checked={task.done}
        onCheckedChange={(v) => onToggleComplete(task.id, Boolean(v))}
        aria-label={`Complete: ${task.title}`}
        className="size-5 shrink-0"
      />

      <span
        className="grid size-7 shrink-0 place-items-center rounded-md"
        style={{ backgroundColor: `hsl(var(--pillar-${task.pillar}) / 0.14)` }}
        aria-hidden
      >
        <KindIcon
          className="size-4"
          style={{ color: `hsl(var(--pillar-${task.pillar}))` }}
        />
      </span>

      <div className="min-w-0 flex-1">
        <p className={cn("truncate text-sm font-medium", task.done && "line-through")}>
          {task.title}
        </p>
        <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
          <Badge variant="outline" className="border-0 px-1.5 py-0 text-2xs"
            style={{ color: `hsl(var(--pillar-${task.pillar}))` }}>
            {task.pillarLabel}
          </Badge>
          <span className="tabular-nums">{task.estMinutes}m</span>
          {task.difficulty && (
            <Badge variant={DIFF_COLOR[task.difficulty]} className="px-1.5 py-0 text-2xs">
              {task.difficulty}
            </Badge>
          )}
          {task.isRevision && (
            <span className="flex items-center gap-1 text-warning">
              <RotateCcw className="size-3" /> Revise
            </span>
          )}
        </div>
      </div>

      <ConfidencePicker
        value={task.confidence}
        pillar={task.pillar}
        onChange={(c) => onSetConfidence(task.id, c)}
        className="hidden group-hover:flex data-[has-value=true]:flex"
        data-has-value={Boolean(task.confidence)}
      />

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" className="size-7 opacity-0 group-hover:opacity-100"
            aria-label="Task actions">
            <MoreHorizontal className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onSelect={() => onEdit(task.id)}>Edit (e)</DropdownMenuItem>
          <DropdownMenuItem onSelect={() => onReschedule(task.id)}>Reschedule (r)</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </motion.li>
  );
}
```

---

*End of design system v1.0. This document is binding; propose changes via PR with a
visual diff and an a11y check. Companion docs: `01-PRD.md` (product), `02-SRS.md`
(functional), `04-DATABASE-SCHEMA.md` (data).*
