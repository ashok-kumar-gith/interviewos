"use client";

import * as React from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { ExternalLink, RefreshCw, Search } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Input } from "@/components/ui/input";
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
  listCompanies,
  listPatterns,
  listProblems,
  type ListResult,
  type Problem,
} from "@/lib/api/content";
import type { Difficulty } from "@/lib/api/types";

const PAGE_SIZE = 20;
const DIFFICULTIES: Difficulty[] = ["easy", "medium", "hard"];

export default function ProblemsPage() {
  const [page, setPage] = React.useState(1);
  const [searchInput, setSearchInput] = React.useState("");
  const [q, setQ] = React.useState("");
  const [difficulty, setDifficulty] = React.useState<Difficulty | "">("");
  const [patternId, setPatternId] = React.useState("");
  const [companyId, setCompanyId] = React.useState("");

  // Debounce free-text search and reset to page 1 on any filter change.
  React.useEffect(() => {
    const id = setTimeout(() => {
      setQ(searchInput.trim());
      setPage(1);
    }, 300);
    return () => clearTimeout(id);
  }, [searchInput]);

  const patternsQuery = useQuery({
    queryKey: ["patterns"],
    queryFn: () => listPatterns(),
  });
  const companiesQuery = useQuery({
    queryKey: ["companies", "catalog"],
    queryFn: () => listCompanies(),
  });

  const query = useQuery<ListResult<Problem>, unknown>({
    queryKey: ["problems", { page, q, difficulty, patternId, companyId }],
    queryFn: () =>
      listProblems({
        page,
        page_size: PAGE_SIZE,
        q: q || undefined,
        difficulty: difficulty || undefined,
        pattern_id: patternId || undefined,
        company_id: companyId || undefined,
      }),
    placeholderData: keepPreviousData,
  });

  const problems = query.data?.data ?? [];
  const meta = query.data?.meta;

  function resetPageThen(setter: () => void) {
    setter();
    setPage(1);
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-h1">Problems</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Browse the DSA catalog — filter by difficulty, pattern, or company.
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
            aria-label="Search problems"
            placeholder="Search problems…"
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
          label="Pattern"
          value={patternId}
          onChange={(v) => resetPageThen(() => setPatternId(v))}
          disabled={patternsQuery.isLoading}
          options={[
            { value: "", label: "All" },
            ...(patternsQuery.data?.data ?? []).map((p) => ({ value: p.id, label: p.name })),
          ]}
        />

        <FilterSelect
          label="Company"
          value={companyId}
          onChange={(v) => resetPageThen(() => setCompanyId(v))}
          disabled={companiesQuery.isLoading}
          options={[
            { value: "", label: "All" },
            ...(companiesQuery.data?.data ?? []).map((c) => ({ value: c.id, label: c.name })),
          ]}
        />
      </div>

      {query.isError ? (
        <Alert variant="danger" title="Couldn't load problems">
          Something went wrong. Try again.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => query.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      ) : query.isLoading ? (
        <Skeleton className="h-80 w-full" />
      ) : problems.length === 0 ? (
        <EmptyState
          icon={Search}
          title="No problems found"
          description="Try clearing a filter or searching for something else."
        />
      ) : (
        <Card className="overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Title</TableHead>
                <TableHead className="w-28">Difficulty</TableHead>
                <TableHead className="w-24 text-right">Est. min</TableHead>
                <TableHead className="w-28 text-right">Frequency</TableHead>
                <TableHead className="w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {problems.map((p) => (
                <TableRow key={p.id}>
                  <TableCell className="font-medium">
                    <span className="flex items-center gap-2">
                      {p.title}
                      {p.is_premium && (
                        <span className="text-2xs uppercase text-warning">Premium</span>
                      )}
                    </span>
                  </TableCell>
                  <TableCell>
                    <DifficultyPill difficulty={p.difficulty} />
                  </TableCell>
                  <TableCell className="text-right tabular-nums text-muted-foreground">
                    {p.estimated_minutes ?? "—"}
                  </TableCell>
                  <TableCell className="text-right tabular-nums text-muted-foreground">
                    {p.frequency_score != null ? p.frequency_score.toFixed(1) : "—"}
                  </TableCell>
                  <TableCell>
                    {p.url ? (
                      <a
                        href={p.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        aria-label={`Open ${p.title} externally`}
                        className="inline-flex text-muted-foreground hover:text-foreground"
                      >
                        <ExternalLink className="size-4" aria-hidden />
                      </a>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      {!query.isLoading && !query.isError && problems.length > 0 && (
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
