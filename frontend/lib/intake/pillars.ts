import type { PillarKey } from "@/lib/nav";

/**
 * Pillar self-assessment definitions for the intake wizard (Step 3).
 *
 * `type` is the API enum value (PillarType in docs/openapi.yaml) used as the
 * key in `pillar_strengths`. `key` is the design-system accent key used to tint
 * UI (`--pillar-*`). Plain data only — no functions — so this is safe to import
 * from both Server and Client components.
 */

/** API pillar enum (PillarType). */
export type PillarType =
  | "dsa"
  | "system_design"
  | "lld"
  | "backend_engineering"
  | "behavioral"
  | "resume";

export interface PillarDef {
  /** API enum value — the key written into `pillar_strengths`. */
  type: PillarType;
  /** Design-system accent key (`--pillar-${key}`). */
  key: PillarKey;
  /** Human label. */
  label: string;
  /** Coach-like one-liner describing what a low/high score means. */
  hint: string;
}

/** Fixed order, matching the design system's pillar order. */
export const PILLARS: PillarDef[] = [
  {
    type: "dsa",
    key: "dsa",
    label: "Data Structures & Algorithms",
    hint: "Arrays to graphs — comfort solving problems under time pressure.",
  },
  {
    type: "system_design",
    key: "system-design",
    label: "System Design",
    hint: "Designing scalable, reliable distributed systems end to end.",
  },
  {
    type: "lld",
    key: "lld",
    label: "Low-Level Design",
    hint: "Class modeling, design patterns, and clean OO abstractions.",
  },
  {
    type: "backend_engineering",
    key: "backend",
    label: "Backend Engineering",
    hint: "Databases, APIs, concurrency, and production depth.",
  },
  {
    type: "behavioral",
    key: "behavioral",
    label: "Behavioral",
    hint: "STAR stories on leadership, conflict, and impact.",
  },
  {
    type: "resume",
    key: "resume",
    label: "Resume",
    hint: "An impact-driven, ATS-friendly resume for the target role.",
  },
];

/** Labels for each 1–5 self-rating, shown under the segmented control. */
export const CONFIDENCE_LABELS: Record<number, string> = {
  1: "New to this",
  2: "Some exposure",
  3: "Comfortable",
  4: "Strong",
  5: "Interview-ready",
};
