/**
 * Mock interview API layer — CRUD, findings, and ranked weaknesses summary.
 * Shapes mirror the OpenAPI schemas: MockInterview, MockInterviewUpsert,
 * MockInterviewDetail, MockFinding, MockFindingUpsert, MockWeaknessSummary.
 */

import { api } from "@/lib/api/client";
import type { PillarType } from "@/lib/api/types";

export type MockType = "coding" | "system_design" | "lld" | "behavioral" | "backend_engineering";
export type MockOutcome =
  | "strong_hire"
  | "hire"
  | "lean_hire"
  | "no_hire"
  | "strong_no_hire"
  | "not_rated";
export type FindingSeverity = "info" | "minor" | "major" | "blocker";

export const MOCK_TYPES: { value: MockType; label: string }[] = [
  { value: "coding", label: "Coding" },
  { value: "system_design", label: "System Design" },
  { value: "lld", label: "LLD" },
  { value: "behavioral", label: "Behavioral" },
  { value: "backend_engineering", label: "Backend Engineering" },
];

export const MOCK_OUTCOMES: { value: MockOutcome; label: string }[] = [
  { value: "strong_hire", label: "Strong Hire" },
  { value: "hire", label: "Hire" },
  { value: "lean_hire", label: "Lean Hire" },
  { value: "no_hire", label: "No Hire" },
  { value: "strong_no_hire", label: "Strong No Hire" },
  { value: "not_rated", label: "Not rated" },
];

export const FINDING_SEVERITIES: { value: FindingSeverity; label: string }[] = [
  { value: "info", label: "Info" },
  { value: "minor", label: "Minor" },
  { value: "major", label: "Major" },
  { value: "blocker", label: "Blocker" },
];

export function mockTypeLabel(t: MockType): string {
  return MOCK_TYPES.find((m) => m.value === t)?.label ?? t;
}

export function mockOutcomeLabel(o: MockOutcome): string {
  return MOCK_OUTCOMES.find((m) => m.value === o)?.label ?? o;
}

export function severityLabel(s: FindingSeverity): string {
  return FINDING_SEVERITIES.find((m) => m.value === s)?.label ?? s;
}

export interface MockInterview {
  id: string;
  user_id: string;
  type: MockType;
  topic_id?: string | null;
  design_problem_id?: string | null;
  company_id?: string | null;
  scheduled_at?: string | null;
  conducted_at?: string | null;
  duration_minutes?: number | null;
  outcome: MockOutcome;
  overall_score?: number | null;
  interviewer?: string | null;
  summary?: string | null;
  created_at?: string;
}

export interface MockFinding {
  id: string;
  mock_interview_id: string;
  pillar_type?: PillarType | null;
  topic_id?: string | null;
  severity: FindingSeverity;
  category: string;
  detail: string;
  remediation_task_id?: string | null;
}

export interface MockInterviewDetail extends MockInterview {
  transcript_md?: string | null;
  findings?: MockFinding[];
}

export interface MockInterviewUpsert {
  type: MockType;
  topic_id?: string;
  design_problem_id?: string;
  company_id?: string;
  scheduled_at?: string;
  conducted_at?: string;
  duration_minutes?: number;
  outcome?: MockOutcome;
  overall_score?: number;
  interviewer?: string;
  transcript_md?: string;
  summary?: string;
}

export interface MockFindingUpsert {
  pillar_type?: PillarType;
  topic_id?: string;
  severity: FindingSeverity;
  category: string;
  detail: string;
  create_remediation_task?: boolean;
}

export interface MockWeaknessItem {
  area: string;
  pillar?: PillarType | null;
  count: number;
  score: number;
  max_severity: FindingSeverity;
  severity_counts: Record<string, number>;
}

export interface MockWeaknessSummary {
  items: MockWeaknessItem[];
  total_findings: number;
  generated_by: string;
}

/** GET /mock-interviews — list past mocks (optionally by type). */
export function listMocks(type?: MockType): Promise<MockInterview[]> {
  return api
    .getList<MockInterview>("/mock-interviews", {
      query: { page_size: 100, type, sort: "-conducted_at" },
    })
    .then((r) => r.data);
}

/** GET /mock-interviews/{id} — mock with findings. */
export function getMock(id: string): Promise<MockInterviewDetail> {
  return api.get<MockInterviewDetail>(`/mock-interviews/${id}`);
}

/** POST /mock-interviews. */
export function createMock(payload: MockInterviewUpsert): Promise<MockInterview> {
  return api.post<MockInterview>("/mock-interviews", payload);
}

/** PUT /mock-interviews/{id}. */
export function updateMock(id: string, payload: MockInterviewUpsert): Promise<MockInterview> {
  return api.put<MockInterview>(`/mock-interviews/${id}`, payload);
}

/** DELETE /mock-interviews/{id}. */
export function deleteMock(id: string): Promise<void> {
  return api.delete<void>(`/mock-interviews/${id}`);
}

/** POST /mock-interviews/{id}/findings — add a finding. */
export function addMockFinding(id: string, payload: MockFindingUpsert): Promise<MockFinding> {
  return api.post<MockFinding>(`/mock-interviews/${id}/findings`, payload);
}

/** GET /mock-interviews/weaknesses — ranked weakness summary. */
export function getMockWeaknesses(): Promise<MockWeaknessSummary> {
  return api.get<MockWeaknessSummary>("/mock-interviews/weaknesses");
}
