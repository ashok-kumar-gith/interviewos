/**
 * Maps the API `PillarType` enum (underscore form, e.g. `system_design`) to the
 * design-system `PillarKey` (hyphen form, e.g. `system-design`) used for accent
 * tokens (`--pillar-*`) and nav icons, plus human labels. Plain data only.
 */

import type { PillarType } from "@/lib/api/types";
import type { PillarKey } from "@/lib/nav";

interface PillarMeta {
  key: PillarKey;
  label: string;
}

const PILLAR_META: Record<PillarType, PillarMeta> = {
  dsa: { key: "dsa", label: "DSA" },
  system_design: { key: "system-design", label: "System Design" },
  lld: { key: "lld", label: "LLD" },
  backend_engineering: { key: "backend", label: "Backend Eng" },
  behavioral: { key: "behavioral", label: "Behavioral" },
  resume: { key: "resume", label: "Resume" },
};

export function pillarKey(type: PillarType): PillarKey {
  return PILLAR_META[type]?.key ?? "dsa";
}

export function pillarLabel(type: PillarType): string {
  return PILLAR_META[type]?.label ?? type;
}
