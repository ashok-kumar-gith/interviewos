# InterviewOS — Cross-Document Alignment Decisions (ADR-000)

**Status:** Accepted · **Date:** 2026-06-29

This document records the canonical resolutions for inconsistencies found in the
Phase-1 planning suite consistency audit. **All other documents must conform to
the decisions below.** Where an audit found two docs in conflict, the decision
names the single source of truth and the required edits.

Authority precedence (unless a decision overrides):
- **Data model / enums / fields:** `04-DATABASE-SCHEMA.md`
- **API surface (paths, methods, DTOs, errors, pagination):** `docs/openapi.yaml`
- **Algorithms (revision, readiness):** `02-SRS.md` §6 (normative spec)
- **Product scope / priorities:** `01-PRD.md`

---

## Data model

**D1 — Revision recall is BINARY.** GA uses `correct | incorrect` everywhere
(PRD §7.13 and SRS §6 are already binary; the schema/OpenAPI SM-2 4-value scale
is removed for GA).
- Schema: rename enum type `recall_quality` → **`recall_result`** with values
  **`('correct','incorrect')`**; `revision_items.last_recall recall_result`.
  Keep `ease NUMERIC(4,2) DEFAULT 2.5` but mark **reserved / inert at GA** (stored,
  never mutated). Keep `stage`, `is_active`, `review_count`, `lapse_count`.
- OpenAPI: rename schema `RecallQuality` → **`RecallResult`** `enum: [correct, incorrect]`;
  `RecallRequest.recall` and `RevisionItem.last_recall` → `$ref RecallResult`;
  `RevisionItem.ease` description: "reserved (inert at GA)".

**D2 — RevisionItem field names follow the schema.** SRS §6.1 adopts schema field
names: **`stage`** (not `interval_index`), **`is_active`** boolean (graduated ⇒
`is_active=false`; remove the `status (active|graduated)` field), plus
`interval_days`, `ease` (reserved/inert), `review_count`, `lapse_count`. Behavior
remains as SRS specifies. Use the term **"graduated"** (≡ `is_active=false`); do
not use "mastered".

**D4 — `mock_type` has 5 values:** `coding, system_design, lld, behavioral,
backend_engineering`. Update PRD §7.12 and SRS FR-MOCK-001 / §1.3 to list all five
(schema/OpenAPI already correct).

**D5 — Confidence is an integer 1–5 (not a string enum).** Schema: store as
`SMALLINT NOT NULL CHECK (col BETWEEN 1 AND 5)`; **remove `confidence_level` from
the ENUM list** (it is a numeric domain). OpenAPI already models `integer, 1..5`.

**D6 — `story_theme` has 10 values:** the 8 in PRD/SRS **plus `ambiguity` and
`impact`**. Update PRD §7.10 and SRS FR-BHV-002 (schema/OpenAPI already have 10).

**D7 — `plan_item_type` per-table domains are documented.** In schema §4: state the
allowed `item_type` subset per table — `plan_tasks` may target
`topic|subtopic|problem|resource|design_problem|lld_problem|behavioral_story|revision_item`;
`revision_items` may target `topic|problem|design_problem|lld_problem`. Reconcile the
entity-overview line so it **includes `subtopic`**. Note in OpenAPI `PlanItemType`
description that the valid subset is context-dependent.

**D8 — `email_verified` is derived.** The API exposes a boolean `email_verified`
derived from `users.email_verified_at`. Keep the boolean in OpenAPI; add a one-line
note in the schema doc (and the OpenAPI `User.email_verified` description) that it
is derived from the timestamp column. No type change.

**D10 — Plan-task initial status term is `pending`.** SRS FR-ROAD-003 must use
`pending` (the `task_status` enum value), not "not-started".

---

## API surface

**D11 — Canonical paths = `openapi.yaml`.** All prose docs must use these exact
resources/paths (the `/api/v1` prefix lives in `servers:`; write paths prefix-less):

| Concern | Canonical | Replaces (stale) |
|---|---|---|
| Register | `POST /auth/register` | `/auth/signup` |
| OAuth | `GET /auth/oauth/{provider}/callback` | `/oauth/{google,github}`, `POST /auth/oauth/callback` |
| Password | `POST /auth/forgot-password`, `POST /auth/reset-password` | `/forgot`, `/reset` |
| Profile | `GET /profile`, `PUT /profile` | `POST/GET/PUT /me/profile` |
| Roadmap (plural) | `POST /roadmaps/generate` (regenerate = body flag), `GET /roadmaps/active`, `GET /roadmaps/{roadmapId}/weeks/{weekNumber}` | `/roadmap/*`, `/roadmap/regenerate`, `/roadmap/weeks/{id}` |
| Plan day | `GET /plan-days/{date}` | `/plan-days/{id}/tasks` (no such collection) |
| Task lifecycle | `POST /tasks/{taskId}/complete`, `POST /tasks/{taskId}/skip`, `POST /tasks/{taskId}/reschedule` (all **POST**) | `PATCH /plan-tasks/{id}`, `/plan-tasks/{id}/complete` |
| DSA | `GET /problems`, `GET /problems/{problemId}`, `GET /patterns` | `/dsa/problems`, `/dsa/patterns` |
| Revision | `POST /revision/{revisionItemId}/recall` | `/revision/{id}/review` |
| Today | `GET /today` | — |
| Company target | `GET/PUT /company/target` (documented singleton sub-resource) | — |

Resource-name rule: the table is `plan_tasks` but the **route resource is `tasks`**;
roadmaps are **plural**. Update `07-ROADMAP.md` §3 API rows and the path examples in
`03-ARCHITECTURE.md` accordingly.

**D12 — Add `GET /dashboard`.** The dashboard is served by a single aggregate
endpoint (the p95 < 2s SLO target), not by the client fanning out to `/today` +
`/analytics/*`. Add to `openapi.yaml` (tag **Analytics**, bearer-auth) and to the
`05-API-CONTRACTS.md` endpoint table. Response schema **`DashboardResponse`**:

```yaml
DashboardResponse:
  type: object
  required: [overall_readiness, pillar_readiness, study_streak, today, revision_due_count, generated_at]
  properties:
    overall_readiness: { type: number, format: float, minimum: 0, maximum: 100 }
    estimated_readiness_date: { type: [string, 'null'], format: date }
    pillar_readiness:
      type: array
      items:
        type: object
        required: [pillar, readiness, coverage, avg_confidence, revision_health]
        properties:
          pillar: { $ref: '#/components/schemas/PillarType' }
          readiness: { type: number, format: float }
          coverage: { type: number, format: float }
          avg_confidence: { type: number, format: float }
          revision_health: { type: number, format: float }
    study_streak:
      type: object
      required: [current, longest]
      properties:
        current: { type: integer }
        longest: { type: integer }
    today:
      type: object
      required: [date, total_tasks, completed_tasks, estimated_hours, remaining_hours]
      properties:
        date: { type: string, format: date }
        total_tasks: { type: integer }
        completed_tasks: { type: integer }
        estimated_hours: { type: number, format: float }
        remaining_hours: { type: number, format: float }
    revision_due_count: { type: integer }
    generated_at: { type: string, format: date-time }
```

**D9 — Notes/Attachments REST API is P1 (deferred).** At GA, per-task notes are
captured via the `notes` field on `POST /tasks/{taskId}/complete`. Polymorphic
`/notes` and `/attachments` endpoints (FR-ROAD-009, **P1**) are out of the GA API
surface; the `owner_type` enum is reserved for them. Document this in
`05-API-CONTRACTS.md`. (Also note `study_sessions` is derived server-side from task
completions at GA — no standalone session endpoint.)

**D13 — Pagination/filter conventions.** Params: `page`, `page_size`, `sort`
(comma-list, `-` prefix = desc), `filter` (RHS-colon: `filter=difficulty:hard,priority:high`),
`q` (search). **Offset pagination only** (no cursor). Fix `03-ARCHITECTURE.md`:
remove the bracket filter syntax (`filter[difficulty]=`) and any `next_cursor`.

**D14 — Error taxonomy canonical = `05-API-CONTRACTS.md` §1.5.** Codes:
`BAD_REQUEST, UNAUTHENTICATED, INVALID_CREDENTIALS, REFRESH_TOKEN_INVALID,
FORBIDDEN, NOT_FOUND, CONFLICT, VALIDATION_ERROR, RATE_LIMITED, AI_UNAVAILABLE,
INTERNAL`. Envelope: `error{code, message, request_id, details[].{field, message}}`.
Fix `03-ARCHITECTURE.md`: complete the code list (or defer to 05 §1.5) and fix the
error example to **snake_case** (`"field":"hours_per_week","message":"…"`).

---

## Algorithms

**D3 — Revision graduation/ease in `03-ARCHITECTURE.md` aligns to SRS.** Interval
ladder `[1,3,7,15,30]`. Graduate when `stage == len(LADDER)-1 && recall == correct`
⇒ `is_active = false`. **Remove the `ease >= MASTERY` gate** and the `MASTERY`
constant; treat `ease` as stored-but-inert at GA. Use `graduated` / `is_active=false`
(not `mastered`).

**D15 — Readiness model canonical = SRS §6.2 (multiplicative).** Fix
`03-ARCHITECTURE.md` §8.1 to:

```
confidence_p   = (avg_rating_p - 1) / 4          # rating ∈ [1..5] → [0..1]
readiness_p    = 100 × coverage_p × (0.6·confidence_p + 0.4·revhealth_p)
# mock blend, once ≥1 mock exists for the pillar (w_mock: 0 → 0.2):
readiness_p    = (1 - w_mock)·readiness_p + w_mock·(100·mock_score_p)
overall        = Σ pillar_weight_p × readiness_p
```

Coverage is a **multiplicative gate** (0 coverage ⇒ 0 readiness). Drop the
3-weight additive `w_cov + w_conf + w_rev` form and the `avgConfidence/5`
normalization. **Estimated readiness date:** daily readiness-gain rate over a
trailing **14-day** window; if `rate <= 0` ⇒ date is undefined ("insufficient
momentum"); target threshold **80**.

---

## Structure

**D16 — OpenAPI source of truth = `backend/api/openapi.yaml`** (contract-first,
maintained with the backend / `swag` annotations). **`docs/openapi.yaml`** is the
committed, published snapshot used for planning and rendered docs. Update `README.md`
and the `03-ARCHITECTURE.md` monorepo tree to show both with these roles; keep
`07-ROADMAP.md` references to `backend/api/openapi.yaml`.

**D17 — Canonical backend module list** (`backend/internal/<module>`):
`auth, user, intake, curriculum, roadmap, progress, content, dsa, systemdesign,
backendeng, lld, designproblems, behavioral, resume, mock, revision, analytics,
company, ai, notification`.
Platform code: `internal/platform/{server, middleware, database}` + `pkg/config`.
Binaries: `cmd/{api, migrate, seed}`. Seed **data** under `backend/seed/<module>/*.yaml`.
Naming rules: **`systemdesign`** (not `sysdesign`); **`notification`** (singular);
`backendeng` (package) maps to enum `backend_engineering` and token `--pillar-backend`.
Update the folder trees in `03-ARCHITECTURE.md` §4 and `07-ROADMAP.md` §0 plus
roadmap file-touch columns to this list.

**D18 — Infra layout.** `infra/docker-compose.yml` (compose lives under `infra/`,
not repo root). `Makefile` at repo root. Dockerfiles at `backend/Dockerfile` and
`frontend/Dockerfile`. `infra/{nginx, k8s, helm}`. **MailHog** is a dev-only compose
service (a local mail catcher — it does **not** contradict the in-app-only GA
notification default). Terraform is optional and, if mentioned, lives under
`infra/terraform/`. Reconcile `README.md`, `03-ARCHITECTURE.md`, and `07-ROADMAP.md`
to this.

---

## Consistent (no change needed)
NFR/SLO targets (dashboard p95 < 2s, API read p99 < 300ms, crash-free ≥ 99.5%);
PRD §13 open-question defaults; feature↔milestone coverage; auth public/protected
sets; the spaced-repetition interval ladder & reset behavior; the four worked
request/response examples in `05-API-CONTRACTS.md`.
