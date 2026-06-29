import type { Metadata } from "next";
import { Gauge } from "lucide-react";
import { ReadinessCard } from "@/components/dashboard/readiness-card";
import { PILLAR_NAV, type PillarKey } from "@/lib/nav";

export const metadata: Metadata = {
  title: "Dashboard",
};

interface PillarReadiness {
  pillar: PillarKey;
  value: number;
  deltaPct: number;
  takeaway?: string;
}

// Static placeholder data — wired to the analytics API in a later milestone.
const PILLAR_DATA: PillarReadiness[] = [
  { pillar: "dsa", value: 74, deltaPct: 6, takeaway: "Strongest pillar — keep momentum." },
  {
    pillar: "system-design",
    value: 41,
    deltaPct: -3,
    takeaway: "System Design is your weakest pillar.",
  },
  { pillar: "lld", value: 58, deltaPct: 2 },
  { pillar: "backend", value: 63, deltaPct: 4 },
  { pillar: "behavioral", value: 52, deltaPct: 1 },
  { pillar: "resume", value: 80, deltaPct: 0, takeaway: "ATS-ready." },
];

const PILLAR_ICON = Object.fromEntries(
  PILLAR_NAV.map((n) => [n.pillar, { icon: n.icon, label: n.label }]),
) as Record<PillarKey, { icon: (typeof PILLAR_NAV)[number]["icon"]; label: string }>;

export default function DashboardPage() {
  return (
    <div className="space-y-8">
      <header>
        <h1 className="text-h1">Dashboard</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          You&apos;re 68% ready · Est. interview-ready Aug 14
        </p>
      </header>

      <section aria-labelledby="overall-readiness">
        <h2 id="overall-readiness" className="sr-only">
          Overall readiness
        </h2>
        <ReadinessCard
          label="Overall"
          value={68}
          deltaPct={4}
          takeaway="On track — System Design is your next focus."
          icon={Gauge}
          variant="overall"
        />
      </section>

      <section aria-labelledby="pillar-readiness" className="space-y-4">
        <h2 id="pillar-readiness" className="text-h3">
          Readiness by pillar
        </h2>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {PILLAR_DATA.map((p) => {
            const meta = PILLAR_ICON[p.pillar];
            return (
              <ReadinessCard
                key={p.pillar}
                label={meta.label}
                value={p.value}
                deltaPct={p.deltaPct}
                takeaway={p.takeaway}
                icon={meta.icon}
                pillar={p.pillar}
                variant="pillar"
              />
            );
          })}
        </div>
      </section>
    </div>
  );
}
