"use client";

import * as React from "react";
import { useMutation } from "@tanstack/react-query";
import { Play, Terminal } from "lucide-react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { ApiError } from "@/lib/api/client";
import { runCode, type CodeLanguage, type RunResult } from "@/lib/api/coderun";
import { cn } from "@/lib/utils";

/** Allowlist of languages, paired with a friendly label and a starter snippet. */
const LANGUAGES: { value: CodeLanguage; label: string; starter: string }[] = [
  { value: "python", label: "Python", starter: "print(1 + 1)\n" },
  { value: "javascript", label: "JavaScript", starter: "console.log(1 + 1);\n" },
  { value: "typescript", label: "TypeScript", starter: "const x: number = 1 + 1;\nconsole.log(x);\n" },
  { value: "go", label: "Go", starter: 'package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println(1 + 1)\n}\n' },
  { value: "java", label: "Java", starter: "public class Main {\n  public static void main(String[] args) {\n    System.out.println(1 + 1);\n  }\n}\n" },
  { value: "cpp", label: "C++", starter: "#include <iostream>\nint main() {\n  std::cout << 1 + 1 << std::endl;\n}\n" },
  { value: "c", label: "C", starter: '#include <stdio.h>\nint main() {\n  printf("%d\\n", 1 + 1);\n}\n' },
];

const MONO = "font-mono text-sm leading-relaxed";

export interface CodeRunnerProps {
  /** Initial language (defaults to python). */
  defaultLanguage?: CodeLanguage;
  /** Initial source; defaults to the language's starter snippet. */
  defaultSource?: string;
  className?: string;
}

/**
 * CodeRunner — a self-contained panel to run a code snippet against the backend
 * executor and view stdout/stderr/exit. Picks a language, edits source + stdin,
 * and runs via a mutation. Resilient to backend/upstream failures: a clear error
 * is shown rather than crashing.
 */
export function CodeRunner({ defaultLanguage = "python", defaultSource, className }: CodeRunnerProps) {
  const starterFor = React.useCallback(
    (lang: CodeLanguage) => LANGUAGES.find((l) => l.value === lang)?.starter ?? "",
    [],
  );

  const [language, setLanguage] = React.useState<CodeLanguage>(defaultLanguage);
  const [source, setSource] = React.useState<string>(defaultSource ?? starterFor(defaultLanguage));
  const [stdin, setStdin] = React.useState<string>("");

  const mutation = useMutation<RunResult, ApiError, void>({
    mutationFn: () => runCode({ language, source, stdin: stdin || undefined }),
  });

  const onLanguageChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const next = e.target.value as CodeLanguage;
    // Swap to the new language's starter only if the editor still holds the
    // current language's untouched starter (don't clobber the user's edits).
    setSource((cur) => (cur === starterFor(language) ? starterFor(next) : cur));
    setLanguage(next);
  };

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    mutation.mutate();
  };

  const result = mutation.data;

  return (
    <Card className={cn("w-full", className)}>
      <CardHeader className="flex-row items-center justify-between gap-3">
        <CardTitle className="flex items-center gap-2">
          <Terminal aria-hidden className="size-4 text-muted-foreground" />
          Code Runner
        </CardTitle>
        <div className="w-44">
          <label htmlFor="code-runner-language" className="sr-only">
            Language
          </label>
          <Select
            id="code-runner-language"
            value={language}
            onChange={onLanguageChange}
            disabled={mutation.isPending}
          >
            {LANGUAGES.map((l) => (
              <option key={l.value} value={l.value}>
                {l.label}
              </option>
            ))}
          </Select>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <label htmlFor="code-runner-source" className="text-sm font-medium">
              Source
            </label>
            <Textarea
              id="code-runner-source"
              value={source}
              onChange={(e) => setSource(e.target.value)}
              spellCheck={false}
              rows={12}
              className={cn(MONO, "resize-y")}
              placeholder="Write code here…"
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="code-runner-stdin" className="text-sm font-medium">
              Stdin <span className="font-normal text-muted-foreground">(optional)</span>
            </label>
            <Textarea
              id="code-runner-stdin"
              value={stdin}
              onChange={(e) => setStdin(e.target.value)}
              spellCheck={false}
              rows={3}
              className={cn(MONO, "resize-y")}
              placeholder="Input passed to the program's stdin…"
            />
          </div>

          <div className="flex items-center gap-3">
            <Button type="submit" loading={mutation.isPending} disabled={source.trim() === ""}>
              {!mutation.isPending && <Play aria-hidden />}
              Run
            </Button>
            {result?.version && !mutation.isPending && (
              <span className="text-xs text-muted-foreground">
                {result.language} · {result.version}
              </span>
            )}
          </div>
        </form>

        <OutputArea pending={mutation.isPending} error={mutation.error} result={result} />
      </CardContent>
    </Card>
  );
}

function OutputArea({
  pending,
  error,
  result,
}: {
  pending: boolean;
  error: ApiError | null;
  result?: RunResult;
}) {
  if (pending) {
    return (
      <div
        aria-live="polite"
        className="flex items-center gap-2 rounded-md border border-border bg-muted/40 px-3.5 py-3 text-sm text-muted-foreground"
      >
        Running…
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="danger" title="Could not run code">
        {error.message}
      </Alert>
    );
  }

  if (!result) return null;

  const exitOk = result.exit_code === 0;

  return (
    <div className="space-y-3" aria-live="polite">
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <span
          className={cn(
            "inline-flex items-center rounded-full px-2 py-0.5 font-medium",
            exitOk ? "bg-success/10 text-success" : "bg-danger/10 text-danger",
          )}
        >
          exit {result.exit_code}
        </span>
        {result.message && <span className="text-muted-foreground">{result.message}</span>}
      </div>

      <OutputBlock label="stdout" content={result.stdout} empty="(no output)" />
      {result.stderr && <OutputBlock label="stderr" content={result.stderr} tone="danger" />}
    </div>
  );
}

function OutputBlock({
  label,
  content,
  empty,
  tone = "default",
}: {
  label: string;
  content: string;
  empty?: string;
  tone?: "default" | "danger";
}) {
  const isEmpty = content === "";
  return (
    <div className="space-y-1">
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
      <pre
        className={cn(
          "max-h-72 overflow-auto rounded-md border px-3.5 py-3 whitespace-pre-wrap break-words",
          MONO,
          tone === "danger"
            ? "border-danger/30 bg-danger/5 text-danger"
            : "border-border bg-muted/40 text-foreground",
        )}
      >
        {isEmpty ? <span className="text-muted-foreground">{empty ?? "(empty)"}</span> : content}
      </pre>
    </div>
  );
}
