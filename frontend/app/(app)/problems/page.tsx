"use client";

import * as React from "react";
import Link from "next/link";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { CheckCircle2, Circle, CircleDashed, ExternalLink, RefreshCw, Search } from "lucide-react";

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
import { NewContentButton } from "@/components/authoring/new-content-button";
import {
  listCompanies,
  listPatterns,
  listProblems,
  type ListResult,
  type Problem,
} from "@/lib/api/content";
import type { Difficulty } from "@/lib/api/types";
import { listProblemProgress } from "@/lib/api/dsaprogress";

const PAGE_SIZE = 20;
const DIFFICULTIES: Difficulty[] = ["easy", "medium", "hard"];

export default function ProblemsPage() {
  const [page, setPage] = React.useState(1);
  const [searchInput, setSearchInput] = React.useState("");
  const [q, setQ] = React.useState("");
  const [difficulty, setDifficulty] = React.useState<Difficulty | "">("");
  const [patternId, setPatternId] = React.useState("");
  const [companyId, setCompanyId] = React.useState("");
  const [status, setStatus] = React.useState<"" | "solved" | "attempted" | "unsolved">("");

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

  // Per-problem progress, to badge rows and filter by status. Best-effort: if it
  // fails (e.g. not logged in) every problem reads as "unsolved" and no badges
  // show. A problem is SOLVED when its row has solved=true; ATTEMPTED when a row
  // exists but isn't solved (opened/worked-on, e.g. saved code or in_progress);
  // UNSOLVED when there is no row at all.
  const progressQuery = useQuery({
    queryKey: ["problems-solved"],
    queryFn: listProblemProgress,
    refetchOnMount: "always",
  });
  const { solvedIds, attemptedIds } = React.useMemo(() => {
    const solved = new Set<string>();
    const attempted = new Set<string>();
    for (const p of progressQuery.data ?? []) {
      if (p.solved) solved.add(p.problem_id);
      else attempted.add(p.problem_id);
    }
    return { solvedIds: solved, attemptedIds: attempted };
  }, [progressQuery.data]);

  function statusOf(id: string): "solved" | "attempted" | "unsolved" {
    if (solvedIds.has(id)) return "solved";
    if (attemptedIds.has(id)) return "attempted";
    return "unsolved";
  }

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

  const allProblems = query.data?.data ?? [];
  // Status filter is applied client-side over the current page (the catalog API
  // has no progress dimension). "All" shows everything.
  const problems = status ? allProblems.filter((p) => statusOf(p.id) === status) : allProblems;
  const meta = query.data?.meta;

  function resetPageThen(setter: () => void) {
    setter();
    setPage(1);
  }

  return (
    <div className="space-y-6">
      <header className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-h1">Problems</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Browse the DSA catalog — filter by difficulty, pattern, or company.
          </p>
        </div>
        <NewContentButton type="problem" label="New problem" />
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
          label="Status"
          value={status}
          onChange={(v) => setStatus(v as typeof status)}
          options={[
            { value: "", label: "All" },
            { value: "solved", label: "Solved" },
            { value: "attempted", label: "Attempted" },
            { value: "unsolved", label: "Unsolved" },
          ]}
        />

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
                      <StatusMark status={statusOf(p.id)} />
                      <Link
                        href={`/problems/${p.id}`}
                        className="rounded-sm underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      >
                        {p.title}
                      </Link>
                      {statusOf(p.id) === "solved" && (
                        <span className="rounded-full bg-success/10 px-2 py-0.5 text-2xs font-medium text-success">
                          Solved
                        </span>
                      )}
                      {statusOf(p.id) === "attempted" && (
                        <span className="rounded-full bg-warning/10 px-2 py-0.5 text-2xs font-medium text-warning">
                          Attempted
                        </span>
                      )}
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

/** Leading status glyph: filled check (solved), dashed ring (attempted), faint
 *  ring (unsolved). Color is paired with the badge text so it's not the sole cue. */
function StatusMark({ status }: { status: "solved" | "attempted" | "unsolved" }) {
  if (status === "solved") {
    return <CheckCircle2 className="size-4 shrink-0 text-success" aria-label="Solved" />;
  }
  if (status === "attempted") {
    return <CircleDashed className="size-4 shrink-0 text-warning" aria-label="Attempted" />;
  }
  return <Circle className="size-4 shrink-0 text-muted-foreground/40" aria-label="Not started" />;
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
