import { BookOpen, Code, FileText, Mic, Play, RotateCcw, type LucideIcon } from "lucide-react";
import type { TaskKind } from "@/lib/api/types";

const KIND_ICON: Record<TaskKind, LucideIcon> = {
  study: BookOpen,
  solve: Code,
  read: FileText,
  watch: Play,
  revise: RotateCcw,
  mock: Mic,
};

const KIND_LABEL: Record<TaskKind, string> = {
  study: "Study",
  solve: "Solve",
  read: "Read",
  watch: "Watch",
  revise: "Revise",
  mock: "Mock",
};

/** Kind icon for a plan task (DESIGN-SYSTEM §4.4 kind icon set). */
export function KindIcon({ kind, className }: { kind: TaskKind; className?: string }) {
  const Icon = KIND_ICON[kind] ?? BookOpen;
  return <Icon className={className} aria-label={KIND_LABEL[kind] ?? kind} />;
}

export function kindLabel(kind: TaskKind): string {
  return KIND_LABEL[kind] ?? kind;
}
