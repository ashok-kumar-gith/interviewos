"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";

import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip } from "@/components/ui/tooltip";
import { getTimeSpent, type TimeSpentResponse } from "@/lib/api/analytics";
import { cn } from "@/lib/utils";

const WEEKS = 13; // ~12 weeks of history, aligned to whole weeks
const DAY_MS = 24 * 60 * 60 * 1000;
const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

/** Local YYYY-MM-DD key (avoids UTC off-by-one vs. the API's day buckets). */
function dayKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
    d.getDate(),
  ).padStart(2, "0")}`;
}

/** 5 intensity buckets from minutes studied (0..4). */
function bucket(minutes: number): number {
  if (minutes <= 0) return 0;
  if (minutes < 30) return 1;
  if (minutes < 60) return 2;
  if (minutes < 120) return 3;
  return 4;
}

const BUCKET_BG = [
  "bg-muted",
  "bg-success/30",
  "bg-success/55",
  "bg-success/80",
  "bg-success",
];

function formatMinutes(min: number): string {
  if (min <= 0) return "No study";
  const h = Math.floor(min / 60);
  const m = min % 60;
  if (h && m) return `${h}h ${m}m`;
  if (h) return `${h}h`;
  return `${m}m`;
}

function formatDate(d: Date): string {
  return `${MONTHS[d.getMonth()]} ${d.getDate()}`;
}

export function StreakHeatmap() {
  // Window: align the end to the upcoming Saturday so each column is a full week.
  const { start, columns } = React.useMemo(() => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const end = new Date(today.getTime() + (6 - today.getDay()) * DAY_MS); // Saturday
    const startDate = new Date(end.getTime() - (WEEKS * 7 - 1) * DAY_MS);
    const cols: Date[][] = [];
    for (let w = 0; w < WEEKS; w++) {
      const col: Date[] = [];
      for (let d = 0; d < 7; d++) {
        col.push(new Date(startDate.getTime() + (w * 7 + d) * DAY_MS));
      }
      cols.push(col);
    }
    return { start: startDate, columns: cols };
  }, []);

  const query = useQuery<TimeSpentResponse, unknown>({
    queryKey: ["time-spent", "heatmap", dayKey(start)],
    queryFn: () => getTimeSpent({ from: dayKey(start), group_by: "day" }),
  });

  const minutesByDay = React.useMemo(() => {
    const map = new Map<string, number>();
    for (const b of query.data?.buckets ?? []) {
      // The API may key buckets as full ISO timestamps or YYYY-MM-DD; normalize.
      const key = b.key.slice(0, 10);
      map.set(key, (map.get(key) ?? 0) + b.minutes);
    }
    return map;
  }, [query.data]);

  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const todayKey = dayKey(today);

  const { activeDays, totalMinutes } = React.useMemo(() => {
    let days = 0;
    let total = 0;
    for (const [, m] of minutesByDay) {
      if (m > 0) days += 1;
      total += m;
    }
    return { activeDays: days, totalMinutes: total };
  }, [minutesByDay]);

  // Month labels: show a month name above the first column where its month begins.
  const monthLabels = columns.map((col, i) => {
    const first = col[0];
    if (i === 0) return MONTHS[first.getMonth()];
    const prevFirst = columns[i - 1][0];
    return first.getMonth() !== prevFirst.getMonth() ? MONTHS[first.getMonth()] : "";
  });

  return (
    <Card className="p-5">
      <div className="mb-4 flex items-baseline justify-between gap-2">
        <h2 className="text-h3 font-semibold">Study activity</h2>
        <span className="text-xs text-muted-foreground">Last 12 weeks</span>
      </div>

      {query.isLoading ? (
        <Skeleton className="h-32 w-full" />
      ) : query.isError ? (
        <p className="text-sm text-muted-foreground">Couldn&apos;t load your activity.</p>
      ) : (
        <>
          <p className="sr-only" role="status">
            You studied on {activeDays} {activeDays === 1 ? "day" : "days"} in the last 12 weeks,
            for a total of {formatMinutes(totalMinutes)}.
          </p>

          <div className="overflow-x-auto">
            <div className="inline-flex flex-col gap-1.5">
              {/* Month labels */}
              <div className="ml-7 flex gap-1" aria-hidden>
                {monthLabels.map((label, i) => (
                  <span key={i} className="w-3 text-2xs text-muted-foreground">
                    {label}
                  </span>
                ))}
              </div>

              <div className="flex gap-1">
                {/* Weekday labels (M / W / F) */}
                <div className="mr-1 flex w-6 flex-col gap-1" aria-hidden>
                  {WEEKDAYS.map((wd, i) => (
                    <span
                      key={wd}
                      className="h-3 text-2xs leading-3 text-muted-foreground"
                    >
                      {i === 1 || i === 3 || i === 5 ? wd[0] : ""}
                    </span>
                  ))}
                </div>

                {/* Week columns */}
                {columns.map((col, ci) => (
                  <div key={ci} className="flex flex-col gap-1">
                    {col.map((d) => {
                      const key = dayKey(d);
                      const isFuture = key > todayKey;
                      const minutes = minutesByDay.get(key) ?? 0;
                      const b = bucket(minutes);
                      const label = `${formatDate(d)}: ${formatMinutes(minutes)}`;
                      if (isFuture) {
                        return (
                          <span
                            key={key}
                            className="size-3 rounded-[3px] bg-muted/30"
                            aria-hidden
                          />
                        );
                      }
                      return (
                        <Tooltip key={key} content={label}>
                          <span
                            role="img"
                            aria-label={label}
                            tabIndex={0}
                            className={cn(
                              "size-3 rounded-[3px] outline-none ring-ring focus-visible:ring-2",
                              key === todayKey && "ring-1 ring-primary",
                              BUCKET_BG[b],
                            )}
                          />
                        </Tooltip>
                      );
                    })}
                  </div>
                ))}
              </div>

              {/* Legend */}
              <div className="ml-7 mt-1 flex items-center gap-1.5 text-2xs text-muted-foreground">
                <span>Less</span>
                {BUCKET_BG.map((bg, i) => (
                  <span key={i} className={cn("size-3 rounded-[3px]", bg)} aria-hidden />
                ))}
                <span>More</span>
              </div>
            </div>
          </div>
        </>
      )}
    </Card>
  );
}
