# InterviewOS — Developer Guide

A working orientation for contributors. For the full design rationale see
`docs/03-ARCHITECTURE.md`; for the build sequence see `docs/07-ROADMAP.md`; for
local setup see `docs/SETUP.md`.

---

## Architecture in one paragraph

InterviewOS is a **modular monolith**: one deployable Go backend (Gin + GORM)
partitioned into strongly-bounded domain modules, fronted by a Next.js SPA, with
PostgreSQL as the system of record and Redis for cache/sessions/rate-limiting.
Each module follows **clean / hexagonal architecture** — a service (use-case)
layer depends only on ports (Go interfaces); HTTP transport, GORM repositories,
the Claude client, and the mailer are adapters injected at composition time
(`cmd/api/main.go`). The deterministic Curriculum, Revision, and Analytics
engines are the source of truth; **AI is optional augmentation** behind a port
with a deterministic fallback.

```
transport (Gin) → service (use-case) → repository (GORM) → Postgres
                       ↑ ports ↑              Redis (cache/session/rate-limit)
```

Dependencies point inward. The domain/service layer never imports Gin or GORM.

---

## Module layout

```
backend/
├── cmd/{api,migrate,seed}/      # entrypoints (composition root, migration runner, seeder)
├── internal/
│   ├── platform/{server,middleware,database,config,logger}/   # cross-cutting
│   ├── auth user intake content  # auth, accounts, onboarding, content spine
│   ├── curriculum roadmap progress     # engine + plan persistence + progress
│   ├── dsa systemdesign backendeng lld designproblems  # pillars
│   ├── behavioral resume mock          # pillars / tooling
│   ├── revision analytics company      # engines + company mode
│   ├── ai notification                 # AI orchestration + notifications
│   └── seed/                           # seed loader internals
├── pkg/                          # domain-agnostic libs (never imports internal/)
├── migrations/                   # NNNNNN_name.up.sql / .down.sql
└── api/openapi.yaml              # contract source of truth
```

Each `internal/<module>` package owns: `entity.go`, `service.go`,
`repository.go` (interface), `repository_gorm.go` (adapter), `handler.go`,
`dto.go`, `routes.go`, and `*_test.go`.

**Dependency rules** (enforced by review + `go vet`):

1. A module imports another module only through its exported service interface
   (port) — never its repository. Cross-module wiring happens in `cmd/api`.
2. `pkg/` is leaf: it never imports `internal/`.
3. The service layer never imports Gin or GORM. GORM lives only in
   `*_repository_gorm.go`; Gin only in `handler.go` / `platform/server`.
4. No import cycles.

The 20+ modules: `auth, user, intake, content, curriculum, roadmap, progress,
dsa, systemdesign, backendeng, lld, designproblems, behavioral, resume, mock,
revision, analytics, company, ai, notification` (+ `platform`, `seed`).

---

## Build / test / run commands

| Task | Command |
|------|---------|
| Boot full stack (Docker) | `make dev` |
| Sudo-free stack | `scripts/dev-local.sh {start-db\|backend\|frontend\|...}` |
| Apply migrations | `make migrate` (`go run ./cmd/migrate up`) |
| Seed content | `make seed` (idempotent) |
| Backend unit tests | `make be-test` |
| Backend integration tests | `make be-test-integration` (`go test ./... -p 1`) |
| Vet | `make be-lint` (`go vet ./...`) |
| Frontend typecheck/lint/build | `make fe-typecheck` / `fe-lint` / `fe-build` |
| Metrics / Swagger | `make metrics` / `make swagger` |

---

## Adding a feature — the 7 stages

Every feature ships through all seven stages **in order** before the next begins
(`docs/07-ROADMAP.md` §1.2). No TODOs in merged code.

1. **Design** — short design note / ADR: domain-model delta, interfaces, sequence,
   edge cases, acceptance criteria traced to PRD §7.
2. **Database** — paired `NNNNNN_name.up.sql` / `.down.sql` migration(s) +
   idempotent seed + repository tests. The number is the next zero-padded
   sequence (current head is `000014`; the next is `000015`). Migrations are
   immutable once merged — corrections go in a new migration.
3. **Backend** — service layer (business rules), domain entities, validation,
   typed errors. Depend on repository **interfaces**, not concrete types.
4. **API** — Gin handlers, DTOs, routing, middleware. **Update
   `backend/api/openapi.yaml` in the same PR**; Swagger UI stays live at
   `/swagger`. (See OpenAPI-first workflow below.)
5. **Frontend** — route segment + feature folder wired via React Query hooks;
   loading/error/empty states; optimistic updates where the PRD forbids a full
   reload (e.g. Today completion).
6. **Tests** — unit (service), repository, integration (handler + db), frontend
   component + e2e.
7. **Documentation** — feature doc/ADR, README/OpenAPI updates, changelog.

Definition of Done (§1.3): migrations up+down + idempotent seed applied in CI;
no TODOs; unit + integration + repo tests pass to the coverage gate; OpenAPI
updated and linting; frontend wired with React Query; a11y + dark mode pass; CI
green.

---

## OpenAPI-first workflow

- `backend/api/openapi.yaml` is the **contract source of truth** (maintained with
  the backend; `swag` annotations on handlers). `docs/openapi.yaml` is a published
  snapshot used for planning/rendered docs.
- The frontend's typed client (`lib/api/client.ts`) is generated from the spec
  (`openapi-typescript`), so a contract change breaks the frontend build on drift.
- Update the spec **in the same PR** as the handler. CI validates it with
  `openapi-spec-validator`.

---

## Testing strategy

**Backend**

- **Unit (service layer):** business rules in isolation; the engines (curriculum,
  revision, analytics) are pure/deterministic and are heavily table-tested +
  property-tested. Run with `make be-test` / `go test ./...`.
- **Repository tests:** run against a real Postgres; cover queries, constraints,
  soft-delete, and migration up/down round-trips.
- **Integration (handler + db):** spin the router, exercise endpoints end-to-end
  against a seeded DB; assert status, DTO shape, and authz.

> **The `-p 1` rule.** Integration and repository tests share a single Postgres
> database. Go runs test **packages** in parallel by default, which would let
> packages race on the same tables (truncations, fixtures, sequences). Run them
> serially with `go test ./... -p 1`. CI does exactly this in the
> `backend-integration` job (Postgres + Redis service containers, migrate, then
> `-p 1`); pure unit tests stay parallel in the `backend` job. Locally:
> `make be-test-integration`.

**Frontend**

- Component tests (vitest + Testing Library) incl. loading/error/empty, a11y
  (axe), dark-mode render.
- E2E (Playwright) for core journeys (signup → intake → roadmap → complete-today,
  revision review, company switch, dashboard).

**Coverage gates** (CI, block merge): backend overall ≥ 80%; engines
(curriculum/revision/analytics) ≥ 90%; frontend feature logic ≥ 75%. Coverage
artifacts are uploaded by CI (`backend-coverage`, `backend-integration-coverage`).

---

## Conventions

- **12-factor config** via `pkg/config`, validated at startup (fail fast).
- **Structured JSON logs** to stdout with `request_id` correlation.
- **Error envelope:** `{ "error": { "code", "message", "details", "request_id" } }`;
  success `{ "data", "meta" }`. Codes from `pkg/apierror`.
- **Pagination:** offset only — `?page=&page_size=&sort=&filter=&q=`, `meta` block
  with `total/page/page_size`.
- **Migrations** are forward-only history; every `.up.sql` has a matching
  `.down.sql`; seed data is versioned and idempotent.
- **Polymorphic plan tasks:** `PlanTask(item_type, item_id, kind)` powers one
  unified Today list across heterogeneous content.
- **No TODOs** in merged code; clean architecture; SOLID.

---

## Where things live

| Need | Location |
|------|----------|
| Wire a new module / DI | `backend/cmd/api/main.go` |
| Add a migration | `backend/migrations/NNNNNN_name.{up,down}.sql` |
| Add seed data | `backend/seed/<module>/` + `cmd/seed` |
| API contract | `backend/api/openapi.yaml` (snapshot: `docs/openapi.yaml`) |
| Middleware (auth, rate-limit, etc.) | `internal/platform/middleware/` |
| Reusable libs | `backend/pkg/` |
| Frontend feature | `frontend/features/<name>/` + `frontend/app/(app)/<route>/` |
| Typed API client | `frontend/lib/api/` |
| Infra (compose/k8s/helm/nginx) | `infra/` |
| CI | `.github/workflows/ci.yml` |
| Deploy guide | `docs/DEPLOYMENT.md` |
