/**
 * Shared helpers for the admin authoring forms. Plain functions + types — no
 * React — so they can be imported by any of the per-type form components.
 */

import { z } from "zod";
import type { Difficulty, PillarType, Priority, ProblemSourceName } from "@/lib/api/types";

/** The kinds of content the authoring UI can create. */
export type ContentType = "problem" | "design-problem" | "lld-problem" | "topic";

export const CONTENT_TYPE_LABEL: Record<ContentType, string> = {
  problem: "DSA problem",
  "design-problem": "System Design problem",
  "lld-problem": "LLD problem",
  topic: "Topic",
};

export const DIFFICULTIES: Difficulty[] = ["easy", "medium", "hard"];
export const PRIORITIES: Priority[] = ["low", "medium", "high", "critical"];

export const PROBLEM_PLATFORMS = [
  "leetcode",
  "hackerrank",
  "codeforces",
  "interviewbit",
  "gfg",
  "custom",
] as const;

export const PROBLEM_SOURCES: ProblemSourceName[] = [
  "blind75",
  "neetcode150",
  "grind75",
  "tech_interview_handbook",
  "leetcode_top",
  "striver_sde",
  "custom",
];

/** Pillar options for the topic form (PillarType enum values + labels). */
export const PILLAR_OPTIONS: { value: PillarType; label: string }[] = [
  { value: "dsa", label: "DSA" },
  { value: "system_design", label: "System Design" },
  { value: "lld", label: "LLD" },
  { value: "backend_engineering", label: "Backend Engineering" },
  { value: "behavioral", label: "Behavioral" },
  { value: "resume", label: "Resume" },
];

/** Shared field validators (reused across forms via zodValidate). */
export const slugSchema = z
  .string()
  .min(1, "Slug is required")
  .max(160, "Slug is too long")
  .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, "Use lowercase letters, numbers, and hyphens");

export const titleSchema = z.string().min(1, "Title is required").max(200, "Title is too long");
export const nameSchema = z.string().min(1, "Name is required").max(200, "Name is too long");

/** Auto-derive a URL-safe slug from a title/name. */
export function slugify(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

/** Parse a numeric text input to number | null (empty → null). */
export function parseOptionalNumber(value: string): number | null {
  const t = value.trim();
  if (t === "") return null;
  const n = Number(t);
  return Number.isFinite(n) ? n : null;
}

/** Trim a text field to `string | null` (empty → null so we don't send ""). */
export function orNull(value: string | undefined | null): string | null {
  const t = (value ?? "").trim();
  return t === "" ? null : t;
}

/** The detail route for a freshly created/updated item of the given type. */
export function detailHref(type: ContentType, id: string): string {
  switch (type) {
    case "problem":
      return `/problems/${id}`;
    case "design-problem":
      return `/system-design/${id}`;
    case "lld-problem":
      return `/lld/${id}`;
    case "topic":
      return `/topics/${id}`;
  }
}

/** The catalog route to return to after a delete. */
export function catalogHref(type: ContentType): string {
  switch (type) {
    case "problem":
      return "/problems";
    case "design-problem":
      return "/system-design";
    case "lld-problem":
      return "/lld";
    case "topic":
      return "/resources";
  }
}
