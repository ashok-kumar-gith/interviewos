"use client";

import * as React from "react";
import Link from "next/link";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { RefreshCw, Search } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { DifficultyPill } from "@/components/ui/difficulty-pill";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { PaginationBar } from "@/components/catalog/pagination-bar";
import {
  listBackendEngineeringTopics,
  type ListResult,
  type Topic,
} from "@/lib/api/content";
import type { Difficulty, Priority } from "@/lib/api/types";
import { PAGE_SIZE } from "@/lib/config";

const DIFFICULTIES: Difficulty[] = ["easy", "medium", "hard"];
const PRIORITIES: Priority[] = ["low", "medium", "high", "critical"];

export default function BackendEngineeringPage() {
  const [page, setPage] = React.useState(1);
  const [searchInput, setSearchInput] = React.useState("");
  const [q, setQ] = React.useState("");
  const [difficulty, setDifficulty] = React.useState<Difficulty | "">("");
  const [priority, setPriority] = React.useState<Priority | "">("");

  // Debounce free-text search and reset to page 1 on any filter change.
  React.useEffect(() => {
    const id = setTimeout(() => {
      setQ(searchInput.trim());
      setPage(1);
    }, 300);
    return () => clearTimeout(id);
  }, [searchInput]);

  const query = useQuery<ListResult<Topic>, unknown>({
    queryKey: ["backend-engineering-topics", { page, q, difficulty, priority }],
    queryFn: () =>
      listBackendEngineeringTopics({
        page,
        page_size: PAGE_SIZE,
        q: q || undefined,
        difficulty: difficulty || undefined,
        priority: priority || undefined,
      }),
    placeholderData: keepPreviousData,
  });

  const topics = query.data?.data ?? [];
  const meta = query.data?.meta;

  function resetPageThen(setter: () => void) {
    setter();
    setPage(1);
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-h1">Backend Engineering</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Backend depth topics for SDE3 interviews.
        </p>
      </header>

      {/* Filters */}
      <div className="flex flex-wrap items-end gap-3">
        <div className="relative min-w-[200px] flex-1">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <Input
            aria-label="Search topics"
            placeholder="Search topics…"
            className="pl-9"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
          />
        </div>

        <FilterSelect
          label="Difficulty"
          value={difficulty}
          onChange={(v) => resetPageThen(() => setDifficulty(v as Difficulty | ""))}
          options={[
            { value: "", label: "All" },
            ...DIFFICULTIES.map((d) => ({ value: d, label: d[0].toUpperCase() + d.slice(1) })),
          ]}
        />

        <FilterSelect
          label="Priority"
          value={priority}
          onChange={(v) => resetPageThen(() => setPriority(v as Priority | ""))}
          options={[
            { value: "", label: "All" },
            ...PRIORITIES.map((p) => ({ value: p, label: p[0].toUpperCase() + p.slice(1) })),
          ]}
        />
      </div>

      {query.isError ? (
        <Alert variant="danger" title="Couldn't load topics">
          Something went wrong. Try again.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => query.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      ) : query.isLoading ? (
        <Skeleton className="h-80 w-full" />
      ) : topics.length === 0 ? (
        <EmptyState
          icon={Search}
          title="No topics found"
          description="Try clearing a filter or searching for something else."
        />
      ) : (
        <Card className="overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Topic</TableHead>
                <TableHead className="w-28">Difficulty</TableHead>
                <TableHead className="w-28">Priority</TableHead>
                <TableHead className="w-24 text-right">Est. hrs</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {topics.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="font-medium">
                    <Link
                      href={`/backend-engineering/${t.id}`}
                      className="rounded-sm underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      {t.name}
                    </Link>
                    {t.summary && (
                      <p className="mt-0.5 text-sm font-normal text-muted-foreground">
                        {t.summary}
                      </p>
                    )}
                  </TableCell>
                  <TableCell>
                    <DifficultyPill difficulty={t.difficulty} />
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" size="sm" className="capitalize">
                      {t.priority}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right tabular-nums text-muted-foreground">
                    {t.estimated_hours ?? "—"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      {!query.isLoading && !query.isError && topics.length > 0 && (
        <PaginationBar
          page={page}
          totalPages={meta?.total_pages as number | undefined}
          total={meta?.total as number | undefined}
          disabled={query.isFetching}
          onPageChange={setPage}
        />
      )}
    </div>
  );
}

function FilterSelect({
  label,
  value,
  onChange,
  options,
  disabled,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string }[];
  disabled?: boolean;
}) {
  const id = React.useId();
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-2xs uppercase text-muted-foreground">
        {label}
      </label>
      <select
        id={id}
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        className="h-9 rounded-sm border border-input bg-background px-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
      >
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  );
}
