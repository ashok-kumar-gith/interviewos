"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Mic, Plus, RefreshCw, TrendingDown } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { Dialog } from "@/components/ui/dialog";
import { FormField } from "@/components/ui/form-field";
import { SelectField } from "@/components/ui/select-field";
import { TextareaField } from "@/components/ui/textarea-field";
import {
  addMockFinding,
  createMock,
  FINDING_SEVERITIES,
  getMockWeaknesses,
  listMocks,
  MOCK_OUTCOMES,
  MOCK_TYPES,
  mockOutcomeLabel,
  mockTypeLabel,
  severityLabel,
  type FindingSeverity,
  type MockFindingUpsert,
  type MockInterview,
  type MockInterviewUpsert,
  type MockOutcome,
  type MockType,
  type MockWeaknessSummary,
} from "@/lib/api/mock";

const MOCKS_KEY = ["mock-interviews"] as const;
const WEAKNESS_KEY = ["mock-interviews", "weaknesses"] as const;

export default function MockPage() {
  const queryClient = useQueryClient();
  const [logOpen, setLogOpen] = React.useState(false);
  const [logError, setLogError] = React.useState<string | null>(null);
  const [findingFor, setFindingFor] = React.useState<MockInterview | null>(null);
  const [findingError, setFindingError] = React.useState<string | null>(null);

  const mocksQuery = useQuery<MockInterview[], unknown>({
    queryKey: MOCKS_KEY,
    queryFn: () => listMocks(),
  });

  const weaknessQuery = useQuery<MockWeaknessSummary, unknown>({
    queryKey: WEAKNESS_KEY,
    queryFn: () => getMockWeaknesses(),
  });

  const createMutation = useMutation({
    mutationFn: (payload: MockInterviewUpsert) => createMock(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: MOCKS_KEY });
      void queryClient.invalidateQueries({ queryKey: WEAKNESS_KEY });
      setLogOpen(false);
      setLogError(null);
    },
    onError: () => setLogError("Couldn't log that mock. Check your connection and retry."),
  });

  const findingMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: MockFindingUpsert }) =>
      addMockFinding(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: WEAKNESS_KEY });
      setFindingFor(null);
      setFindingError(null);
    },
    onError: () => setFindingError("Couldn't add that finding. Try again."),
  });

  if (mocksQuery.isLoading) return <MockSkeleton />;

  if (mocksQuery.isError) {
    return (
      <Page onLog={() => setLogOpen(true)}>
        <Alert variant="danger" title="Couldn't load your mocks">
          Something went wrong.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => mocksQuery.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      </Page>
    );
  }

  const mocks = mocksQuery.data ?? [];

  return (
    <Page onLog={() => setLogOpen(true)}>
      {/* Weakness summary */}
      <WeaknessSummary
        query={weaknessQuery}
        onRetry={() => weaknessQuery.refetch()}
      />

      <section className="space-y-3">
        <h2 className="text-h3">Past mocks</h2>
        {mocks.length === 0 ? (
          <EmptyState
            icon={Mic}
            title="No mocks logged yet"
            description="Log a mock interview after each practice round — the findings sharpen your plan."
          >
            <Button className="mt-2" onClick={() => setLogOpen(true)}>
              <Plus aria-hidden /> Log a mock
            </Button>
          </EmptyState>
        ) : (
          <div className="space-y-3">
            {mocks.map((m) => (
              <MockCard key={m.id} mock={m} onAddFinding={() => setFindingFor(m)} />
            ))}
          </div>
        )}
      </section>

      <Dialog
        open={logOpen}
        onClose={() => {
          setLogOpen(false);
          setLogError(null);
        }}
        title="Log a mock interview"
        description="Capture the type, outcome, and notes while it's fresh."
      >
        <LogMockForm
          saving={createMutation.isPending}
          error={logError}
          onSubmit={(payload) => createMutation.mutate(payload)}
          onCancel={() => {
            setLogOpen(false);
            setLogError(null);
          }}
        />
      </Dialog>

      <Dialog
        open={Boolean(findingFor)}
        onClose={() => {
          setFindingFor(null);
          setFindingError(null);
        }}
        title="Add a finding"
        description={findingFor ? mockTypeLabel(findingFor.type) : undefined}
      >
        {findingFor && (
          <AddFindingForm
            saving={findingMutation.isPending}
            error={findingError}
            onSubmit={(payload) =>
              findingMutation.mutate({ id: findingFor.id, payload })
            }
            onCancel={() => {
              setFindingFor(null);
              setFindingError(null);
            }}
          />
        )}
      </Dialog>
    </Page>
  );
}

function Page({ children, onLog }: { children: React.ReactNode; onLog: () => void }) {
  return (
    <div className="space-y-6">
      <header className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-h1">Mock interviews</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Log rounds, capture findings, and see where you&apos;re weakest.
          </p>
        </div>
        <Button onClick={onLog}>
          <Plus aria-hidden /> Log a mock
        </Button>
      </header>
      {children}
    </div>
  );
}

const OUTCOME_VARIANT: Record<MockOutcome, "success" | "info" | "warning" | "danger" | "secondary"> = {
  strong_hire: "success",
  hire: "success",
  lean_hire: "info",
  no_hire: "warning",
  strong_no_hire: "danger",
  not_rated: "secondary",
};

function MockCard({ mock, onAddFinding }: { mock: MockInterview; onAddFinding: () => void }) {
  const when = mock.conducted_at ?? mock.scheduled_at ?? mock.created_at;
  return (
    <Card>
      <CardContent className="space-y-2 p-5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span
              className="grid size-7 place-items-center rounded-md"
              style={{ backgroundColor: "hsl(var(--primary) / 0.14)" }}
              aria-hidden
            >
              <Mic className="size-4 text-primary" />
            </span>
            <div>
              <p className="font-medium leading-tight">{mockTypeLabel(mock.type)}</p>
              {when && <p className="text-xs text-muted-foreground">{formatDate(when)}</p>}
            </div>
          </div>
          <div className="flex items-center gap-2">
            {mock.overall_score != null && (
              <span className="text-sm font-semibold tabular-nums">
                {Math.round(mock.overall_score)}/100
              </span>
            )}
            <Badge variant={OUTCOME_VARIANT[mock.outcome]} size="sm">
              {mockOutcomeLabel(mock.outcome)}
            </Badge>
          </div>
        </div>
        {mock.summary && <p className="text-sm text-muted-foreground">{mock.summary}</p>}
        <div className="flex items-center justify-between pt-1">
          {mock.interviewer ? (
            <span className="text-xs text-muted-foreground">with {mock.interviewer}</span>
          ) : (
            <span />
          )}
          <Button variant="ghost" size="sm" onClick={onAddFinding}>
            <Plus aria-hidden /> Add finding
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

const SEVERITY_VARIANT: Record<FindingSeverity, "secondary" | "info" | "warning" | "danger"> = {
  info: "secondary",
  minor: "info",
  major: "warning",
  blocker: "danger",
};

function WeaknessSummary({
  query,
  onRetry,
}: {
  query: ReturnType<typeof useQuery<MockWeaknessSummary, unknown>>;
  onRetry: () => void;
}) {
  if (query.isLoading) {
    return (
      <Card className="space-y-3 p-5">
        <Skeleton className="h-5 w-40" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-3/4" />
      </Card>
    );
  }
  if (query.isError) {
    return (
      <Alert variant="danger" title="Couldn't load your weakness summary">
        <div className="mt-2">
          <Button variant="outline" size="sm" onClick={onRetry}>
            <RefreshCw aria-hidden /> Retry
          </Button>
        </div>
      </Alert>
    );
  }
  const data = query.data;
  if (!data || data.items.length === 0) {
    return (
      <Card>
        <CardContent className="flex items-center gap-3 p-5">
          <span className="grid size-9 place-items-center rounded-md bg-muted text-muted-foreground">
            <TrendingDown className="size-5" aria-hidden />
          </span>
          <div>
            <p className="font-medium">No weaknesses ranked yet</p>
            <p className="text-sm text-muted-foreground">
              Add findings to your mocks and we&apos;ll rank where to focus.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }
  return (
    <Card>
      <CardContent className="space-y-3 p-5">
        <div className="flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-h3">
            <TrendingDown className="size-5 text-warning" aria-hidden />
            Top weaknesses
          </h2>
          <span className="text-xs text-muted-foreground tabular-nums">
            {data.total_findings} findings · {data.generated_by}
          </span>
        </div>
        <ol className="space-y-2">
          {data.items.map((item, i) => (
            <li
              key={`${item.area}-${i}`}
              className="flex items-center gap-3 rounded-md border border-border px-3 py-2"
            >
              <span className="grid size-6 shrink-0 place-items-center rounded-full bg-muted text-xs font-semibold tabular-nums">
                {i + 1}
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{item.area}</p>
                <p className="text-xs text-muted-foreground tabular-nums">
                  {item.count} {item.count === 1 ? "finding" : "findings"}
                </p>
              </div>
              <Badge variant={SEVERITY_VARIANT[item.max_severity]} size="sm">
                {severityLabel(item.max_severity)}
              </Badge>
            </li>
          ))}
        </ol>
      </CardContent>
    </Card>
  );
}

function LogMockForm({
  saving,
  error,
  onSubmit,
  onCancel,
}: {
  saving: boolean;
  error: string | null;
  onSubmit: (payload: MockInterviewUpsert) => void;
  onCancel: () => void;
}) {
  const [type, setType] = React.useState<MockType>("coding");
  const [outcome, setOutcome] = React.useState<MockOutcome>("not_rated");
  const [scoreText, setScoreText] = React.useState("");
  const [interviewer, setInterviewer] = React.useState("");
  const [summary, setSummary] = React.useState("");
  const [scoreError, setScoreError] = React.useState<string>();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    let overall: number | undefined;
    if (scoreText.trim()) {
      const n = Number(scoreText);
      if (Number.isNaN(n) || n < 0 || n > 100) {
        setScoreError("Score must be between 0 and 100.");
        return;
      }
      overall = n;
    }
    setScoreError(undefined);
    onSubmit({
      type,
      outcome,
      overall_score: overall,
      interviewer: interviewer.trim() || undefined,
      summary: summary.trim() || undefined,
      conducted_at: new Date().toISOString(),
    });
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && <Alert variant="danger">{error}</Alert>}
      <SelectField
        id="mock-type"
        label="Type"
        required
        value={type}
        onChange={(e) => setType(e.target.value as MockType)}
      >
        {MOCK_TYPES.map((t) => (
          <option key={t.value} value={t.value}>
            {t.label}
          </option>
        ))}
      </SelectField>
      <SelectField
        id="mock-outcome"
        label="Outcome"
        required
        value={outcome}
        onChange={(e) => setOutcome(e.target.value as MockOutcome)}
      >
        {MOCK_OUTCOMES.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </SelectField>
      <FormField
        id="mock-score"
        label="Overall score (0–100)"
        type="number"
        min={0}
        max={100}
        inputMode="numeric"
        value={scoreText}
        onChange={(e) => setScoreText(e.target.value)}
        error={scoreError}
      />
      <FormField
        id="mock-interviewer"
        label="Interviewer"
        placeholder='e.g. "Peer · Priya"'
        value={interviewer}
        onChange={(e) => setInterviewer(e.target.value)}
      />
      <TextareaField
        id="mock-summary"
        label="Notes"
        rows={3}
        placeholder="What went well, what to work on."
        value={summary}
        onChange={(e) => setSummary(e.target.value)}
      />
      <div className="flex items-center justify-end gap-2 pt-1">
        <Button type="button" variant="outline" onClick={onCancel} disabled={saving}>
          Cancel
        </Button>
        <Button type="submit" loading={saving}>
          Log mock
        </Button>
      </div>
    </form>
  );
}

function AddFindingForm({
  saving,
  error,
  onSubmit,
  onCancel,
}: {
  saving: boolean;
  error: string | null;
  onSubmit: (payload: MockFindingUpsert) => void;
  onCancel: () => void;
}) {
  const [category, setCategory] = React.useState("");
  const [severity, setSeverity] = React.useState<FindingSeverity>("minor");
  const [detail, setDetail] = React.useState("");
  const [createTask, setCreateTask] = React.useState(false);
  const [categoryError, setCategoryError] = React.useState<string>();
  const [detailError, setDetailError] = React.useState<string>();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    let ok = true;
    if (!category.trim()) {
      setCategoryError("Name the area (e.g. \"Scalability\").");
      ok = false;
    } else setCategoryError(undefined);
    if (!detail.trim()) {
      setDetailError("Describe the finding.");
      ok = false;
    } else setDetailError(undefined);
    if (!ok) return;
    onSubmit({
      category: category.trim(),
      severity,
      detail: detail.trim(),
      create_remediation_task: createTask,
    });
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && <Alert variant="danger">{error}</Alert>}
      <FormField
        id="finding-category"
        label="Area / category"
        required
        placeholder='e.g. "System design depth"'
        value={category}
        onChange={(e) => setCategory(e.target.value)}
        error={categoryError}
      />
      <SelectField
        id="finding-severity"
        label="Severity"
        required
        value={severity}
        onChange={(e) => setSeverity(e.target.value as FindingSeverity)}
      >
        {FINDING_SEVERITIES.map((s) => (
          <option key={s.value} value={s.value}>
            {s.label}
          </option>
        ))}
      </SelectField>
      <TextareaField
        id="finding-detail"
        label="Recommendation"
        required
        rows={3}
        placeholder="What to do about it."
        value={detail}
        onChange={(e) => setDetail(e.target.value)}
        error={detailError}
      />
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          className="size-4 rounded border-border text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          checked={createTask}
          onChange={(e) => setCreateTask(e.target.checked)}
        />
        Add a remediation task to my plan
      </label>
      <div className="flex items-center justify-end gap-2 pt-1">
        <Button type="button" variant="outline" onClick={onCancel} disabled={saving}>
          Cancel
        </Button>
        <Button type="submit" loading={saving}>
          Add finding
        </Button>
      </div>
    </form>
  );
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

function MockSkeleton() {
  return (
    <div className="space-y-6" aria-busy>
      <span className="sr-only" role="status">
        Loading your mocks
      </span>
      <header className="space-y-2">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-64" />
      </header>
      <Card className="space-y-3 p-5">
        <Skeleton className="h-5 w-40" />
        <Skeleton className="h-4 w-full" />
      </Card>
      <div className="space-y-3">
        {[0, 1, 2].map((i) => (
          <Card key={i} className="p-5">
            <Skeleton className="h-5 w-1/3" />
            <Skeleton className="mt-2 h-4 w-2/3" />
          </Card>
        ))}
      </div>
    </div>
  );
}
