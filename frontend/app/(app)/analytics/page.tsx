"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip as RechartsTooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  ArrowDownRight,
  ArrowUpRight,
  BarChart3,
  Clock,
  LineChart as LineChartIcon,
  RefreshCw,
  TrendingUp,
} from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert } from "@/components/ui/alert";
import { EmptyState } from "@/components/ui/empty-state";
import { Button } from "@/components/ui/button";
import {
  getSnapshots,
  getTopicAnalytics,
  getTimeSpent,
  type ReadinessSnapshot,
  type TopicAnalyticsResponse,
  type TopicAnalyticsEntry,
  type TimeSpentResponse,
} from "@/lib/api/analytics";
import { ApiError } from "@/lib/api/client";
import { pillarLabel } from "@/lib/pillar-meta";
import type { PillarType } from "@/lib/api/types";
import { cn } from "@/lib/utils";

const retryNon404 = (count: number, error: unknown) =>
  !(error instanceof ApiError && error.status === 404) && count < 1;

function formatDate(iso: string): string {
  const d = new Date(`${iso.slice(0, 10)}T00:00:00`);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function formatMinutes(min: number): string {
  if (min <= 0) return "0m";
  const h = Math.floor(min / 60);
  const m = Math.round(min % 60);
  if (h && m) return `${h}h ${m}m`;
  if (h) return `${h}h`;
  return `${m}m`;
}

export default function AnalyticsPage() {
  const snapshots = useQuery<ReadinessSnapshot[], unknown>({
    queryKey: ["analytics", "snapshots"],
    queryFn: () => getSnapshots(),
    retry: retryNon404,
  });
  const topics = useQuery<TopicAnalyticsResponse, unknown>({
    queryKey: ["analytics", "topics"],
    queryFn: () => getTopicAnalytics(),
    retry: retryNon404,
  });
  const timeSpent = useQuery<TimeSpentResponse, unknown>({
    queryKey: ["analytics", "time-spent"],
    queryFn: () => getTimeSpent({ group_by: "pillar" }),
    retry: retryNon404,
  });

  if (snapshots.isLoading && topics.isLoading && timeSpent.isLoading) {
    return <AnalyticsSkeleton />;
  }

  return (
    <div className="space-y-8">
      <header>
        <h1 className="text-h1">Analytics</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Your readiness trend, topic mastery, and study time.
        </p>
      </header>

      <section aria-labelledby="readiness-trend" className="space-y-4">
        <h2 id="readiness-trend" className="text-h3">
          Readiness over time
        </h2>
        <ReadinessTrend query={snapshots} />
      </section>

      <section aria-labelledby="topic-mastery" className="space-y-4">
        <h2 id="topic-mastery" className="text-h3">
          Topic mastery
        </h2>
        <TopicMastery query={topics} />
      </section>

      <section aria-labelledby="study-time" className="space-y-4">
        <h2 id="study-time" className="text-h3">
          Study time
        </h2>
        <TimeSpentBreakdown query={timeSpent} />
      </section>
    </div>
  );
}

/* ── Readiness over time ─────────────────────────────────────────── */

function ReadinessTrend({
  query,
}: {
  query: ReturnType<typeof useQuery<ReadinessSnapshot[], unknown>>;
}) {
  if (query.isLoading) return <Skeleton className="h-64 w-full" />;
  if (query.isError) {
    return <SectionError label="readiness trend" onRetry={() => query.refetch()} />;
  }

  const rows = (query.data ?? [])
    .filter((s) => s.snapshot_date)
    .slice()
    .sort((a, b) => a.snapshot_date.localeCompare(b.snapshot_date));

  if (rows.length === 0) {
    return (
      <EmptyState
        icon={LineChartIcon}
        title="No readiness history yet"
        description="As you complete tasks, we'll snapshot your readiness so you can watch the trend climb."
      />
    );
  }

  const chartData = rows.map((s) => ({
    date: formatDate(s.snapshot_date),
    readiness: Math.round(s.overall_readiness),
  }));

  const first = chartData[0].readiness;
  const last = chartData[chartData.length - 1].readiness;
  const delta = last - first;

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between">
        <CardTitle className="text-base">Overall readiness</CardTitle>
        <span
          className={cn(
            "inline-flex items-center gap-1 text-sm font-medium tabular-nums",
            delta > 0 ? "text-success" : delta < 0 ? "text-danger" : "text-muted-foreground",
          )}
        >
          {delta > 0 ? (
            <ArrowUpRight className="size-4" aria-hidden />
          ) : delta < 0 ? (
            <ArrowDownRight className="size-4" aria-hidden />
          ) : (
            <TrendingUp className="size-4" aria-hidden />
          )}
          {delta > 0 ? "+" : ""}
          {delta} pts
        </span>
      </CardHeader>
      <CardContent>
        <div className="h-64 w-full" aria-hidden>
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData} margin={{ top: 8, right: 8, bottom: 0, left: -16 }}>
              <CartesianGrid stroke="hsl(var(--border))" strokeDasharray="3 3" vertical={false} />
              <XAxis
                dataKey="date"
                tick={{ fill: "hsl(var(--muted-foreground))", fontSize: 11 }}
                tickLine={false}
                axisLine={{ stroke: "hsl(var(--border))" }}
                minTickGap={24}
              />
              <YAxis
                domain={[0, 100]}
                tick={{ fill: "hsl(var(--muted-foreground))", fontSize: 11 }}
                tickLine={false}
                axisLine={false}
                width={36}
              />
              <RechartsTooltip
                cursor={{ stroke: "hsl(var(--border))" }}
                contentStyle={{
                  background: "hsl(var(--card))",
                  border: "1px solid hsl(var(--border))",
                  borderRadius: 8,
                  fontSize: 12,
                  color: "hsl(var(--foreground))",
                }}
                formatter={(v: number) => [`${v}%`, "Readiness"]}
              />
              <Line
                type="monotone"
                dataKey="readiness"
                stroke="hsl(var(--primary))"
                strokeWidth={2}
                dot={false}
                isAnimationActive={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
        <table className="sr-only">
          <caption>Overall readiness percentage over time</caption>
          <thead>
            <tr>
              <th>Date</th>
              <th>Readiness</th>
            </tr>
          </thead>
          <tbody>
            {chartData.map((d, i) => (
              <tr key={`${d.date}-${i}`}>
                <td>{d.date}</td>
                <td>{d.readiness}%</td>
              </tr>
            ))}
          </tbody>
        </table>
      </CardContent>
    </Card>
  );
}

/* ── Weak & strong topics ────────────────────────────────────────── */

function TopicMastery({
  query,
}: {
  query: ReturnType<typeof useQuery<TopicAnalyticsResponse, unknown>>;
}) {
  if (query.isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-2">
        <Skeleton className="h-48" />
        <Skeleton className="h-48" />
      </div>
    );
  }
  if (query.isError) {
    return <SectionError label="topic mastery" onRetry={() => query.refetch()} />;
  }

  const weak = query.data?.weak ?? [];
  const strong = query.data?.strong ?? [];

  if (weak.length === 0 && strong.length === 0) {
    return (
      <EmptyState
        icon={BarChart3}
        title="No topic data yet"
        description="Once you've worked through some topics, your strongest and weakest areas will surface here."
      />
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2">
      <TopicList
        title="Needs work"
        emptyText="No weak spots flagged — nice."
        topics={weak}
        tone="danger"
      />
      <TopicList
        title="Strongest"
        emptyText="No strong topics yet."
        topics={strong}
        tone="success"
      />
    </div>
  );
}

function TopicList({
  title,
  emptyText,
  topics,
  tone,
}: {
  title: string;
  emptyText: string;
  topics: TopicAnalyticsEntry[];
  tone: "danger" | "success";
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {topics.length === 0 ? (
          <p className="text-sm text-muted-foreground">{emptyText}</p>
        ) : (
          <ul className="space-y-3">
            {topics.map((t) => {
              const pct = Math.round(t.completion_pct ?? 0);
              return (
                <li key={t.topic_id} className="space-y-1">
                  <div className="flex items-baseline justify-between gap-2">
                    <span className="truncate text-sm font-medium">{t.topic_name}</span>
                    <span className="shrink-0 text-2xs uppercase text-muted-foreground">
                      {pillarLabel(t.pillar_type as PillarType)}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                      <div
                        className={cn(
                          "h-full rounded-full",
                          tone === "success" ? "bg-success" : "bg-danger",
                        )}
                        style={{ width: `${Math.min(100, Math.max(0, pct))}%` }}
                      />
                    </div>
                    <span className="w-10 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
                      {pct}%
                    </span>
                  </div>
                  <p className="text-2xs text-muted-foreground">
                    {t.confidence != null ? `Confidence ${t.confidence}/5` : "No confidence yet"}
                    {t.revision_accuracy != null
                      ? ` · Recall ${Math.round(t.revision_accuracy)}%`
                      : ""}
                  </p>
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

/* ── Time spent ──────────────────────────────────────────────────── */

function TimeSpentBreakdown({
  query,
}: {
  query: ReturnType<typeof useQuery<TimeSpentResponse, unknown>>;
}) {
  if (query.isLoading) return <Skeleton className="h-56 w-full" />;
  if (query.isError) {
    return <SectionError label="study time" onRetry={() => query.refetch()} />;
  }

  const buckets = (query.data?.buckets ?? [])
    .slice()
    .sort((a, b) => b.minutes - a.minutes);
  const total = query.data?.total_minutes ?? buckets.reduce((sum, b) => sum + b.minutes, 0);
  const groupBy = query.data?.group_by ?? "pillar";

  if (buckets.length === 0 || total <= 0) {
    return (
      <EmptyState
        icon={Clock}
        title="No study time logged yet"
        description="Time you spend on tasks and revision will be broken down here by pillar."
      />
    );
  }

  const max = Math.max(...buckets.map((b) => b.minutes), 1);

  const labelFor = (key: string) =>
    groupBy === "pillar" ? pillarLabel(key as PillarType) : formatDate(key);

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between">
        <CardTitle className="text-base">
          By {groupBy === "pillar" ? "pillar" : "day"}
        </CardTitle>
        <span className="inline-flex items-center gap-1 text-sm text-muted-foreground">
          <Clock className="size-4" aria-hidden />
          {formatMinutes(total)} total
        </span>
      </CardHeader>
      <CardContent>
        <ul className="space-y-3">
          {buckets.map((b) => {
            const share = total > 0 ? Math.round((b.minutes / total) * 100) : 0;
            return (
              <li key={b.key} className="space-y-1">
                <div className="flex items-baseline justify-between gap-2">
                  <span className="truncate text-sm font-medium">{labelFor(b.key)}</span>
                  <span className="shrink-0 text-xs tabular-nums text-muted-foreground">
                    {formatMinutes(b.minutes)} · {share}%
                  </span>
                </div>
                <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    className="h-full rounded-full bg-primary"
                    style={{ width: `${Math.round((b.minutes / max) * 100)}%` }}
                  />
                </div>
              </li>
            );
          })}
        </ul>
      </CardContent>
    </Card>
  );
}

/* ── Shared bits ─────────────────────────────────────────────────── */

function SectionError({ label, onRetry }: { label: string; onRetry: () => void }) {
  return (
    <Alert variant="danger" title={`Couldn't load your ${label}`}>
      Something went wrong. Try again.
      <div className="mt-3">
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCw aria-hidden /> Retry
        </Button>
      </div>
    </Alert>
  );
}

function AnalyticsSkeleton() {
  return (
    <div className="space-y-8" aria-busy>
      <span className="sr-only" role="status">
        Loading analytics
      </span>
      <header className="space-y-2">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-4 w-72" />
      </header>
      <Skeleton className="h-64 w-full" />
      <div className="grid gap-4 md:grid-cols-2">
        <Skeleton className="h-48" />
        <Skeleton className="h-48" />
      </div>
      <Skeleton className="h-56 w-full" />
    </div>
  );
}
