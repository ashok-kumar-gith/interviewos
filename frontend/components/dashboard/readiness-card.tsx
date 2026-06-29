"use client";

import { motion, useReducedMotion } from "framer-motion";
import { ArrowDownRight, ArrowUpRight, Gauge } from "lucide-react";
import { ProgressRing } from "@/components/ui/progress-ring";
import { Card } from "@/components/ui/card";
import { PILLAR_NAV, type PillarKey } from "@/lib/nav";
import { cn } from "@/lib/utils";

interface ReadinessCardProps {
  label: string; // "Overall" | "System Design"
  value: number; // 0–100
  deltaPct?: number; // vs last week, signed
  takeaway?: string; // one-line insight
  pillar?: PillarKey; // omit for Overall (uses primary)
  variant?: "overall" | "pillar";
  loading?: boolean;
}

export function ReadinessCard({
  label,
  value,
  deltaPct,
  takeaway,
  pillar,
  variant = "pillar",
  loading = false,
}: ReadinessCardProps) {
  const reduce = useReducedMotion();
  // Resolve the icon on the client so the parent Server Component never has to
  // pass a (non-serializable) component function across the RSC boundary.
  const Icon = (pillar && PILLAR_NAV.find((n) => n.pillar === pillar)?.icon) || Gauge;
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
          showLabel={false}
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
                {isUp ? (
                  <ArrowUpRight className="size-3.5" />
                ) : (
                  <ArrowDownRight className="size-3.5" />
                )}
                {Math.abs(deltaPct)}%
                <span className="sr-only">{isUp ? "up" : "down"} from last week</span>
              </span>
            )}
          </div>
          {takeaway && <p className="mt-1 truncate text-sm text-muted-foreground">{takeaway}</p>}
        </div>
      </Card>
    </motion.div>
  );
}
