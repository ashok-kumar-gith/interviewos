"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Check,
  Download,
  FileText,
  Gauge,
  Pencil,
  Plus,
  RefreshCw,
  Sparkles,
  Trash2,
  Upload,
} from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { Dialog } from "@/components/ui/dialog";
import { FormField } from "@/components/ui/form-field";
import { TextareaField } from "@/components/ui/textarea-field";
import { ProgressRing } from "@/components/ui/progress-ring";
import { ResumeProjectEditor } from "@/components/resume/resume-project-editor";
import { ApiError } from "@/lib/api/client";
import {
  createResumeProject,
  deleteResumeFile,
  deleteResumeProject,
  downloadResumeFile,
  getResumeFileMeta,
  getResumeProfile,
  listResumeProjects,
  scoreResume,
  updateResumeProject,
  uploadResumeFile,
  upsertResumeProfile,
  type ResumeFileMeta,
  type ResumeProfile,
  type ResumeProject,
  type ResumeProjectUpsert,
  type ResumeScoreResponse,
} from "@/lib/api/resume";

const PROFILE_KEY = ["resume", "profile"] as const;
const PROJECTS_KEY = ["resume", "projects"] as const;
const FILE_META_KEY = ["resume", "file", "meta"] as const;

type ProjectEditor =
  | { mode: "closed" }
  | { mode: "create" }
  | { mode: "edit"; project: ResumeProject };

export default function ResumePage() {
  const queryClient = useQueryClient();

  const profileQuery = useQuery<ResumeProfile | null, unknown>({
    queryKey: PROFILE_KEY,
    queryFn: () =>
      getResumeProfile().catch((err) => {
        if (err instanceof ApiError && err.status === 404) return null;
        throw err;
      }),
  });

  const projectsQuery = useQuery<ResumeProject[], unknown>({
    queryKey: PROJECTS_KEY,
    queryFn: () =>
      listResumeProjects().catch((err) => {
        if (err instanceof ApiError && err.status === 404) return [];
        throw err;
      }),
  });

  const [editor, setEditor] = React.useState<ProjectEditor>({ mode: "closed" });
  const [editorError, setEditorError] = React.useState<string | null>(null);
  const [actionError, setActionError] = React.useState<string | null>(null);
  const [scoreError, setScoreError] = React.useState<string | null>(null);
  const [score, setScore] = React.useState<ResumeScoreResponse | null>(null);
  const [scoreOpen, setScoreOpen] = React.useState(false);

  // Profile form state
  const [headline, setHeadline] = React.useState("");
  const [summary, setSummary] = React.useState("");
  const [yearsExp, setYearsExp] = React.useState("");
  const [skillsText, setSkillsText] = React.useState("");
  const [keywordsText, setKeywordsText] = React.useState("");
  const [profileSaved, setProfileSaved] = React.useState(false);

  const profile = profileQuery.data;
  React.useEffect(() => {
    if (!profile) return;
    setHeadline(profile.headline ?? "");
    setSummary(profile.summary ?? "");
    setYearsExp(profile.years_experience != null ? String(profile.years_experience) : "");
    setSkillsText((profile.skills ?? []).join(", "));
    setKeywordsText((profile.target_keywords ?? []).join(", "));
  }, [profile]);

  const saveProfileMutation = useMutation({
    mutationFn: () =>
      upsertResumeProfile({
        headline: headline.trim() || undefined,
        summary: summary.trim() || undefined,
        years_experience: yearsExp.trim() ? Number(yearsExp) : undefined,
        skills: splitCsv(skillsText),
        target_keywords: splitCsv(keywordsText),
      }),
    onMutate: () => {
      setActionError(null);
      setProfileSaved(false);
    },
    onSuccess: () => {
      setProfileSaved(true);
      void queryClient.invalidateQueries({ queryKey: PROFILE_KEY });
    },
    onError: () => setActionError("Couldn't save your profile. Check your connection and retry."),
  });

  const saveProjectMutation = useMutation({
    mutationFn: (payload: ResumeProjectUpsert) =>
      editor.mode === "edit"
        ? updateResumeProject(editor.project.id, payload)
        : createResumeProject(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: PROJECTS_KEY });
      setEditor({ mode: "closed" });
      setEditorError(null);
    },
    onError: () => setEditorError("Couldn't save the project. Try again."),
  });

  const deleteProjectMutation = useMutation({
    mutationFn: (id: string) => deleteResumeProject(id),
    onMutate: () => setActionError(null),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: PROJECTS_KEY }),
    onError: () => setActionError("Couldn't delete the project. Try again."),
  });

  const scoreMutation = useMutation({
    mutationFn: () => scoreResume(),
    onMutate: () => {
      setScoreError(null);
      setScore(null);
      setScoreOpen(true);
    },
    onSuccess: (data) => {
      setScore(data);
      void queryClient.invalidateQueries({ queryKey: PROFILE_KEY });
    },
    onError: (err) =>
      setScoreError(
        err instanceof ApiError && err.status === 404
          ? "Add a resume profile first, then run the ATS check."
          : err instanceof ApiError && err.status === 503
            ? "The scorer is busy right now. Try again in a moment."
            : "Couldn't score your resume. Try again.",
      ),
  });

  if (profileQuery.isLoading || projectsQuery.isLoading) return <ResumeSkeleton />;

  if (profileQuery.isError || projectsQuery.isError) {
    return (
      <Page onScore={() => scoreMutation.mutate()} scoring={false} hasProfile={false}>
        <Alert variant="danger" title="Couldn't load your resume">
          Something went wrong.
          <div className="mt-3">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                void profileQuery.refetch();
                void projectsQuery.refetch();
              }}
            >
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      </Page>
    );
  }

  const projects = projectsQuery.data ?? [];

  return (
    <Page
      onScore={() => scoreMutation.mutate()}
      scoring={scoreMutation.isPending}
      hasProfile
    >
      {actionError && <Alert variant="danger">{actionError}</Alert>}

      {profile?.ats_score != null && (
        <Card>
          <CardContent className="flex items-center gap-4 p-5">
            <ProgressRing
              value={profile.ats_score}
              size={64}
              color={scoreColor(profile.ats_score)}
              ariaLabel={`ATS score ${Math.round(profile.ats_score)} out of 100`}
            />
            <div>
              <p className="text-2xs uppercase text-muted-foreground">Last ATS score</p>
              <p className="text-h3 font-bold tabular-nums">{Math.round(profile.ats_score)}/100</p>
              {profile.last_scored_at && (
                <p className="text-xs text-muted-foreground">
                  Scored {formatDate(profile.last_scored_at)}
                </p>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Resume file */}
      <ResumeFileCard />

      {/* Profile editor */}
      <Card>
        <CardContent className="space-y-4 p-5">
          <h2 className="text-h3">Profile</h2>
          {profileSaved && (
            <Alert variant="success" className="text-sm">
              Profile saved.
            </Alert>
          )}
          <FormField
            id="resume-headline"
            label="Headline"
            placeholder='e.g. "Senior Backend Engineer · Distributed systems"'
            value={headline}
            onChange={(e) => {
              setHeadline(e.target.value);
              setProfileSaved(false);
            }}
          />
          <TextareaField
            id="resume-summary"
            label="Summary"
            rows={3}
            placeholder="Two or three lines on who you are and what you ship."
            value={summary}
            onChange={(e) => {
              setSummary(e.target.value);
              setProfileSaved(false);
            }}
          />
          <FormField
            id="resume-years"
            label="Years of experience"
            type="number"
            min={0}
            inputMode="decimal"
            value={yearsExp}
            onChange={(e) => {
              setYearsExp(e.target.value);
              setProfileSaved(false);
            }}
          />
          <FormField
            id="resume-skills"
            label="Skills"
            placeholder="comma-separated, e.g. Go, Kubernetes, Postgres"
            value={skillsText}
            onChange={(e) => {
              setSkillsText(e.target.value);
              setProfileSaved(false);
            }}
          />
          <FormField
            id="resume-keywords"
            label="Target keywords"
            hint="The ATS keywords for your target role — we check your resume against these."
            placeholder="comma-separated, e.g. microservices, event-driven, SRE"
            value={keywordsText}
            onChange={(e) => {
              setKeywordsText(e.target.value);
              setProfileSaved(false);
            }}
          />
          <div className="flex justify-end">
            <Button onClick={() => saveProfileMutation.mutate()} loading={saveProfileMutation.isPending}>
              Save profile
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Projects */}
      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-h3">Projects</h2>
          <Button variant="outline" size="sm" onClick={() => setEditor({ mode: "create" })}>
            <Plus aria-hidden /> Add project
          </Button>
        </div>
        {projects.length === 0 ? (
          <EmptyState
            icon={FileText}
            title="No projects yet"
            description="Add the projects you'll talk about — impact bullets and the tech behind them."
          />
        ) : (
          <div className="space-y-3">
            {projects.map((p) => (
              <ProjectCard
                key={p.id}
                project={p}
                onEdit={() => setEditor({ mode: "edit", project: p })}
                onDelete={() => deleteProjectMutation.mutate(p.id)}
                deleting={
                  deleteProjectMutation.isPending && deleteProjectMutation.variables === p.id
                }
              />
            ))}
          </div>
        )}
      </section>

      <Dialog
        open={editor.mode !== "closed"}
        onClose={() => {
          setEditor({ mode: "closed" });
          setEditorError(null);
        }}
        title={editor.mode === "edit" ? "Edit project" : "Add project"}
      >
        {editor.mode !== "closed" && (
          <ResumeProjectEditor
            initial={editor.mode === "edit" ? editor.project : undefined}
            saving={saveProjectMutation.isPending}
            error={editorError}
            onSubmit={(payload) => saveProjectMutation.mutate(payload)}
            onCancel={() => {
              setEditor({ mode: "closed" });
              setEditorError(null);
            }}
          />
        )}
      </Dialog>

      <Dialog
        open={scoreOpen}
        onClose={() => setScoreOpen(false)}
        title="ATS check"
        description="Your resume scored against the target keywords."
      >
        {scoreMutation.isPending ? (
          <div className="space-y-3" aria-busy>
            <span className="sr-only" role="status">
              Scoring your resume
            </span>
            <Skeleton className="mx-auto size-24 rounded-full" />
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        ) : scoreError ? (
          <Alert variant="danger">{scoreError}</Alert>
        ) : score ? (
          <ScorePanel score={score} />
        ) : null}
      </Dialog>
    </Page>
  );
}

const ACCEPTED_FILE_TYPES = ".pdf,.docx";

function ResumeFileCard() {
  const queryClient = useQueryClient();
  const inputRef = React.useRef<HTMLInputElement>(null);
  const [error, setError] = React.useState<string | null>(null);

  const metaQuery = useQuery<ResumeFileMeta | null, unknown>({
    queryKey: FILE_META_KEY,
    queryFn: () =>
      getResumeFileMeta().catch((err) => {
        if (err instanceof ApiError && err.status === 404) return null;
        throw err;
      }),
  });

  const uploadMutation = useMutation({
    mutationFn: (file: File) => uploadResumeFile(file),
    onMutate: () => setError(null),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: FILE_META_KEY }),
    onError: (err) =>
      setError(
        err instanceof ApiError && (err.status === 422 || err.status === 400)
          ? err.message || "That file couldn't be uploaded. Use a PDF or DOCX under 5MB."
          : "Couldn't upload your resume file. Try again.",
      ),
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteResumeFile(),
    onMutate: () => setError(null),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: FILE_META_KEY }),
    onError: () => setError("Couldn't delete your resume file. Try again."),
  });

  const downloadMutation = useMutation({
    mutationFn: (fileName: string) => downloadResumeFile(fileName),
    onMutate: () => setError(null),
    onError: () => setError("Couldn't download your resume file. Try again."),
  });

  function onPickFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = ""; // allow re-selecting the same file
    if (file) uploadMutation.mutate(file);
  }

  const meta = metaQuery.data;
  const busy = uploadMutation.isPending;

  return (
    <Card>
      <CardContent className="space-y-4 p-5">
        <h2 className="text-h3">Resume file</h2>
        <p className="text-sm text-muted-foreground">
          Upload your resume as a PDF or DOCX (max 5MB). One file is kept — uploading again replaces it.
        </p>

        {error && <Alert variant="danger">{error}</Alert>}

        <input
          ref={inputRef}
          type="file"
          accept={ACCEPTED_FILE_TYPES}
          className="sr-only"
          onChange={onPickFile}
          aria-label="Choose a resume file to upload"
        />

        {metaQuery.isLoading ? (
          <Skeleton className="h-10 w-full" />
        ) : meta ? (
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-border p-3">
            <div className="flex items-center gap-2 min-w-0">
              <FileText className="size-5 shrink-0 text-muted-foreground" aria-hidden />
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{meta.file_name}</p>
                <p className="text-xs text-muted-foreground">{formatBytes(meta.size_bytes)}</p>
              </div>
            </div>
            <div className="flex items-center gap-1">
              <Button
                variant="outline"
                size="sm"
                onClick={() => downloadMutation.mutate(meta.file_name)}
                loading={downloadMutation.isPending}
              >
                <Download aria-hidden /> Download
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => inputRef.current?.click()}
                loading={busy}
              >
                <Upload aria-hidden /> Replace
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="size-8 text-muted-foreground hover:text-danger"
                onClick={() => deleteMutation.mutate()}
                loading={deleteMutation.isPending}
                aria-label="Delete resume file"
              >
                <Trash2 aria-hidden />
              </Button>
            </div>
          </div>
        ) : (
          <Button variant="outline" onClick={() => inputRef.current?.click()} loading={busy}>
            <Upload aria-hidden /> Upload resume file
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function Page({
  children,
  onScore,
  scoring,
  hasProfile,
}: {
  children: React.ReactNode;
  onScore: () => void;
  scoring: boolean;
  hasProfile: boolean;
}) {
  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <header className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-h1">Resume</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Keep your profile and projects sharp, then run an ATS check.
          </p>
        </div>
        <Button variant="outline" onClick={onScore} loading={scoring} disabled={!hasProfile}>
          <Gauge aria-hidden /> Score / ATS check
        </Button>
      </header>
      {children}
    </div>
  );
}

function ProjectCard({
  project,
  onEdit,
  onDelete,
  deleting,
}: {
  project: ResumeProject;
  onEdit: () => void;
  onDelete: () => void;
  deleting: boolean;
}) {
  return (
    <Card>
      <CardContent className="space-y-2 p-5">
        <div className="flex items-start justify-between gap-2">
          <div>
            <h3 className="font-medium leading-snug">{project.name}</h3>
            {project.role && <p className="text-xs text-muted-foreground">{project.role}</p>}
          </div>
          <div className="flex items-center gap-1">
            <Button variant="ghost" size="sm" onClick={onEdit}>
              <Pencil aria-hidden /> Edit
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="size-7 text-muted-foreground hover:text-danger"
              onClick={onDelete}
              loading={deleting}
              aria-label={`Delete project: ${project.name}`}
            >
              <Trash2 aria-hidden />
            </Button>
          </div>
        </div>
        {project.impact && <p className="text-sm text-muted-foreground">{project.impact}</p>}
        {project.metrics && project.metrics.length > 0 && (
          <ul className="space-y-0.5">
            {project.metrics.map((m, i) => (
              <li key={i} className="flex items-start gap-1.5 text-sm">
                <Check className="mt-0.5 size-3.5 shrink-0 text-success" aria-hidden />
                {m}
              </li>
            ))}
          </ul>
        )}
        {project.tech_stack && project.tech_stack.length > 0 && (
          <div className="flex flex-wrap gap-1 pt-1">
            {project.tech_stack.map((t) => (
              <Badge key={t} variant="secondary" size="sm">
                {t}
              </Badge>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ScorePanel({ score }: { score: ResumeScoreResponse }) {
  const pct = Math.round(score.ats_score);
  return (
    <div className="space-y-5">
      <div className="flex flex-col items-center gap-2">
        <ProgressRing
          value={pct}
          size={96}
          stroke={8}
          color={scoreColor(pct)}
          ariaLabel={`ATS score ${pct} out of 100`}
        />
        <p className="text-sm text-muted-foreground">{scoreVerdict(pct)}</p>
      </div>

      {score.used_fallback && (
        <Alert variant="warning">
          The AI reviewer was unavailable — this is a heuristic ATS estimate.
        </Alert>
      )}

      {(score.keyword_matches?.length || score.missing_keywords?.length) ? (
        <div className="space-y-2">
          <p className="text-2xs uppercase text-muted-foreground">Keyword match</p>
          <div className="flex flex-wrap gap-1">
            {score.keyword_matches?.map((k) => (
              <Badge key={`m-${k}`} variant="success" size="sm">
                <Check className="size-3" aria-hidden /> {k}
              </Badge>
            ))}
            {score.missing_keywords?.map((k) => (
              <Badge key={`x-${k}`} variant="outline" size="sm" className="text-muted-foreground">
                {k}
              </Badge>
            ))}
          </div>
        </div>
      ) : null}

      {score.suggestions && score.suggestions.length > 0 && (
        <div className="space-y-2">
          <p className="text-2xs uppercase text-muted-foreground">Suggestions</p>
          <ul className="space-y-2">
            {score.suggestions.map((s, i) => (
              <li key={i} className="flex items-start gap-2 text-sm">
                <Sparkles
                  className="mt-0.5 size-4 shrink-0"
                  style={{ color: "hsl(var(--pillar-resume))" }}
                  aria-hidden
                />
                <span>{s}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

function scoreColor(score: number): string {
  if (score >= 75) return "hsl(var(--success))";
  if (score >= 50) return "hsl(var(--warning))";
  return "hsl(var(--danger))";
}

function scoreVerdict(score: number): string {
  if (score >= 75) return "Strong — ATS-ready for the target role.";
  if (score >= 50) return "Solid base — a few fixes will lift this.";
  return "Needs work — address the suggestions below.";
}

function splitCsv(text: string): string[] {
  return text
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

function ResumeSkeleton() {
  return (
    <div className="mx-auto max-w-3xl space-y-6" aria-busy>
      <span className="sr-only" role="status">
        Loading your resume
      </span>
      <header className="space-y-2">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-4 w-64" />
      </header>
      <Card className="space-y-3 p-5">
        <Skeleton className="h-5 w-24" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-20 w-full" />
      </Card>
    </div>
  );
}
