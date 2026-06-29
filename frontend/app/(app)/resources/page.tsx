"use client";

import * as React from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Library, RefreshCw, Search } from "lucide-react";

import { Alert } from "@/components/ui/alert";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { ResourceRow } from "@/components/catalog/resource-row";
import { PaginationBar } from "@/components/catalog/pagination-bar";
import { listResources, type ListResult, type Resource } from "@/lib/api/content";
import type { Difficulty, ResourceType } from "@/lib/api/types";

const PAGE_SIZE = 20;

const TYPES: { value: ResourceType; label: string }[] = [
  { value: "book", label: "Book" },
  { value: "video", label: "Video" },
  { value: "article", label: "Article" },
  { value: "course", label: "Course" },
  { value: "github", label: "GitHub" },
  { value: "practice", label: "Practice" },
  { value: "documentation", label: "Docs" },
  { value: "blog", label: "Blog" },
  { value: "cheatsheet", label: "Cheatsheet" },
];

const DIFFICULTIES: Difficulty[] = ["easy", "medium", "hard"];

export default function ResourcesPage() {
  const [page, setPage] = React.useState(1);
  const [searchInput, setSearchInput] = React.useState("");
  const [q, setQ] = React.useState("");
  const [type, setType] = React.useState<ResourceType | "">("");
  const [difficulty, setDifficulty] = React.useState<Difficulty | "">("");

  React.useEffect(() => {
    const id = setTimeout(() => {
      setQ(searchInput.trim());
      setPage(1);
    }, 300);
    return () => clearTimeout(id);
  }, [searchInput]);

  const query = useQuery<ListResult<Resource>, unknown>({
    queryKey: ["resources", { page, q, type, difficulty }],
    queryFn: () =>
      listResources({
        page,
        page_size: PAGE_SIZE,
        q: q || undefined,
        type: type || undefined,
        difficulty: difficulty || undefined,
      }),
    placeholderData: keepPreviousData,
  });

  const resources = query.data?.data ?? [];
  const meta = query.data?.meta;
  const hasFilters = Boolean(q || type || difficulty);

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-h1">Resource Library</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Curated books, videos, courses, and references — filter by type or difficulty.
        </p>
      </header>

      <div className="flex flex-wrap items-end gap-3">
        <div className="relative min-w-[200px] flex-1">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <Input
            aria-label="Search resources"
            placeholder="Search resources…"
            className="pl-9"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
          />
        </div>

        <FilterSelect
          label="Type"
          value={type}
          onChange={(v) => {
            setType(v as ResourceType | "");
            setPage(1);
          }}
          options={[{ value: "", label: "All" }, ...TYPES]}
        />

        <FilterSelect
          label="Difficulty"
          value={difficulty}
          onChange={(v) => {
            setDifficulty(v as Difficulty | "");
            setPage(1);
          }}
          options={[
            { value: "", label: "All" },
            ...DIFFICULTIES.map((d) => ({ value: d, label: d[0].toUpperCase() + d.slice(1) })),
          ]}
        />
      </div>

      {query.isError ? (
        <Alert variant="danger" title="Couldn't load resources">
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
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      ) : resources.length === 0 ? (
        <EmptyState
          icon={hasFilters ? Search : Library}
          title={hasFilters ? "No resources match these filters" : "No resources yet"}
          description={
            hasFilters
              ? "Try clearing a filter or searching for something else."
              : "Curated study resources will appear here as your plan is built."
          }
        />
      ) : (
        <ul className="space-y-2">
          {resources.map((r) => (
            <li key={r.id}>
              <ResourceRow resource={r} />
            </li>
          ))}
        </ul>
      )}

      {!query.isLoading && !query.isError && resources.length > 0 && (
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
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string }[];
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
        onChange={(e) => onChange(e.target.value)}
        className="h-9 rounded-sm border border-input bg-background px-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
