"use client";

import {
  PolarAngleAxis,
  PolarGrid,
  Radar,
  RadarChart,
  ResponsiveContainer,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export interface PillarRadarDatum {
  pillar: string;
  readiness: number;
}

/**
 * PillarRadarChart (DESIGN-SYSTEM §4.2) — the six pillars' readiness on one
 * polygon. Includes a visually-hidden data table fallback for accessibility.
 */
export function PillarRadar({ data }: { data: PillarRadarDatum[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Readiness across pillars</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-52 w-full" aria-hidden>
          <ResponsiveContainer width="100%" height="100%">
            <RadarChart data={data} outerRadius="70%">
              <PolarGrid stroke="hsl(var(--border))" />
              <PolarAngleAxis
                dataKey="pillar"
                tick={{ fill: "hsl(var(--muted-foreground))", fontSize: 11 }}
              />
              <Radar
                dataKey="readiness"
                stroke="hsl(var(--primary))"
                fill="hsl(var(--primary))"
                fillOpacity={0.25}
                isAnimationActive={false}
              />
            </RadarChart>
          </ResponsiveContainer>
        </div>
        <table className="sr-only">
          <caption>Readiness percentage by pillar</caption>
          <thead>
            <tr>
              <th>Pillar</th>
              <th>Readiness</th>
            </tr>
          </thead>
          <tbody>
            {data.map((d) => (
              <tr key={d.pillar}>
                <td>{d.pillar}</td>
                <td>{d.readiness}%</td>
              </tr>
            ))}
          </tbody>
        </table>
      </CardContent>
    </Card>
  );
}
