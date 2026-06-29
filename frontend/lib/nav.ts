import {
  Binary,
  Boxes,
  CalendarRange,
  FileText,
  LayoutDashboard,
  type LucideIcon,
  MessagesSquare,
  Network,
  Server,
  Sparkles,
} from "lucide-react";

export type PillarKey =
  | "dsa"
  | "system-design"
  | "lld"
  | "backend"
  | "behavioral"
  | "resume";

export interface NavItem {
  href: string;
  label: string;
  icon: LucideIcon;
  /** Pillar key drives the accent color/dot; omit for primary (non-pillar) items. */
  pillar?: PillarKey;
}

/** Top primary nav: the daily entry points. */
export const PRIMARY_NAV: NavItem[] = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/today", label: "Today", icon: Sparkles },
  { href: "/roadmap", label: "Roadmap", icon: CalendarRange },
];

/** Pillar nav — fixed order, fixed icons (DESIGN-SYSTEM §2.3 / §9). */
export const PILLAR_NAV: NavItem[] = [
  { href: "/problems", label: "DSA", icon: Binary, pillar: "dsa" },
  { href: "/system-design", label: "System Design", icon: Network, pillar: "system-design" },
  { href: "/lld", label: "LLD", icon: Boxes, pillar: "lld" },
  { href: "/backend-engineering", label: "Backend Eng", icon: Server, pillar: "backend" },
  { href: "/behavioral", label: "Behavioral", icon: MessagesSquare, pillar: "behavioral" },
  { href: "/resume", label: "Resume", icon: FileText, pillar: "resume" },
];
