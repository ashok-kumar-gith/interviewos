"use client";

import * as React from "react";
import Link from "next/link";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { ChevronRight, RefreshCw, Search } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { DifficultyPill } from "@/components/ui/difficulty-pill";
import { PaginationBar } from "@/components/catalog/pagination-bar";
import type { ListResult } from "@/lib/api/content";
import type { Difficulty } from "@/lib/api/types";

const PAGE_SIZE = 20;
const DIFFICULTIES: Difficulty[] = ["easy", "medium", "hard"];

interface OrderedProblem {
  id: string;
  title: string;
  difficulty: Difficulty;
  order_index: number;
}

export interface OrderedProblemListProps<T extends OrderedProblem> {
  title: string;
  subtitle: string;
  queryKey: string;
  /** Base path each row links to — `${hrefBase}/${id}` (e.g. "/system-design"). */
  hrefBase: string;
  fetcher: (filters: {
    page?: number;
    page_size?: number;
    difficulty?: Difficulty;
    q?: string;
  }) => Promise<ListResult<T>>;
}

export function OrderedProblemList<T extends OrderedProblem>({
  title,
  subtitle,
  queryKey,
  hrefBase,
  fetcher,
}: OrderedProblemListProps<T>) {
  const [page, setPage] = React.useState(1);
  const [searchInput, setSearchInput] = React.useState("");
  const [q, setQ] = React.useState("");
  const [difficulty, setDifficulty] = React.useState<Difficulty | "">("");

  React.useEffect(() => {
    const id = setTimeout(() => {
      setQ(searchInput.trim());
      setPage(1);
    }, 300);
    return () => clearTimeout(id);
  }, [searchInput]);

  const query = useQuery<ListResult<T>, unknown>({
    queryKey: [queryKey, { page, q, difficulty }],
    queryFn: () =>
      fetcher({
        page,
        page_size: PAGE_SIZE,
        q: q || undefined,
        difficulty: difficulty || undefined,
      }),
    placeholderData: keepPreviousData,
  });

  const items = query.data?.data ?? [];
  const meta = query.data?.meta;

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-h1">{title}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{subtitle}</p>
      </header>

      <div className="flex flex-wrap items-end gap-3">
        <div className="relative min-w-[200px] flex-1">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <Input
            aria-label={`Search ${title}`}
            placeholder="Search…"
            className="pl-9"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <span className="text-2xs uppercase text-muted-foreground">Difficulty</span>
          <select
            aria-label="Filter by difficulty"
            value={difficulty}
            onChange={(e) => {
              setDifficulty(e.target.value as Difficulty | "");
              setPage(1);
            }}
            className="h-9 rounded-sm border border-input bg-background px-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <option value="">All</option>
            {DIFFICULTIES.map((d) => (
              <option key={d} value={d}>
                {d[0].toUpperCase() + d.slice(1)}
              </option>
            ))}
          </select>
        </div>
      </div>

      {query.isError ? (
        <Alert variant="danger" title="Couldn't load this catalog">
          Something went wrong. Try again.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => query.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      ) : query.isLoading ? (
        <div className="space-y-2">
          {[0, 1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      ) : items.length === 0 ? (
        <EmptyState
          icon={Search}
          title="Nothing found"
          description="Try clearing the difficulty filter or searching for something else."
        />
      ) : (
        <ol className="space-y-2">
          {items.map((item, i) => (
            <li key={item.id}>
              <Link
                href={`${hrefBase}/${item.id}`}
                className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <Card className="flex items-center gap-3 p-4 transition-colors hover:bg-muted/40">
                  <span className="grid size-7 shrink-0 place-items-center rounded-md bg-muted text-xs font-medium tabular-nums text-muted-foreground">
                    {item.order_index ?? (page - 1) * PAGE_SIZE + i + 1}
                  </span>
                  <span className="min-w-0 flex-1 truncate text-sm font-medium">{item.title}</span>
                  <DifficultyPill difficulty={item.difficulty} />
                  <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
                </Card>
              </Link>
            </li>
          ))}
        </ol>
      )}

      {!query.isLoading && !query.isError && items.length > 0 && (
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
