/**
 * Code Runner API layer — execute a snippet server-side and return the result.
 * Shapes mirror the OpenAPI schemas: CodeRunRequest, CodeRunResult.
 */

import { api } from "@/lib/api/client";

/** Languages accepted by the executor allowlist (mirrors the backend). */
export type CodeLanguage = "python" | "javascript" | "typescript" | "go" | "java" | "cpp" | "c";

export interface RunCodeRequest {
  language: CodeLanguage;
  source: string;
  stdin?: string;
}

export interface RunResult {
  stdout: string;
  stderr: string;
  exit_code: number;
  language: string;
  version: string;
  ran: boolean;
  message?: string;
}

/** POST /code/run — run a code snippet and return stdout/stderr/exit. */
export function runCode(body: RunCodeRequest): Promise<RunResult> {
  return api.post<RunResult>("/code/run", body);
}
