# CLAUDE.md — contributor & agent guide

Concise orientation for working in this repo. Deeper detail: `docs/DEVELOPER.md`,
`docs/SETUP.md`, `docs/DEPLOYMENT.md`, `docs/03-ARCHITECTURE.md`,
`docs/07-ROADMAP.md`.

## What this is

InterviewOS — a learning OS for technical-interview prep. **Modular monolith**:
one Go backend (Gin + GORM, clean architecture) + Next.js 15 / React 19 / TS /
Tailwind frontend, Postgres (system of record) + Redis (cache/session/rate-limit).
Deterministic engines (Curriculum, Revision, Analytics) are the source of truth;
AI (Claude) is optional augmentation behind a port with a deterministic fallback.
First track at GA: Backend SDE3. Status: M0–M4 built, M5 hardening in progress.

## Repo layout

```
backend/   cmd/{api,migrate,seed}, internal/<module>, pkg/, migrations/, api/openapi.yaml
frontend/  app/, features/, components/ui/, lib/{api,auth}/, stores/
infra/     docker-compose.yml, nginx/, k8s/ (kustomize), helm/interviewos/
docs/      00–07 numbered planning docs (do not edit lightly) + SETUP/DEPLOYMENT/DEVELOPER
.github/   workflows/ci.yml
Makefile   repo-root task runner
```

Backend modules (20+): `auth user intake content curriculum roadmap progress dsa
systemdesign backendeng lld designproblems behavioral resume mock revision
analytics company ai notification` (+ `platform`, `seed`).

## Build / test / run

```bash
make dev                  # full stack via infra/docker-compose.yml (Docker)
scripts/dev-local.sh ...  # sudo-free ~/.local stack (no Docker): start-db|backend|frontend|migrate|seed|status
make migrate              # go run ./cmd/migrate up
make seed                 # idempotent content seed
make be-test              # go test ./...                (unit; parallel)
make be-test-integration  # go test ./... -p 1           (needs Postgres+Redis; serial)
make be-lint              # go vet ./...
make fe-typecheck / fe-lint / fe-build
make metrics / make swagger
make k8s-apply / k8s-dryrun / helm-template / helm-install
```

Go module: `github.com/interviewos/backend`, Go 1.24. Frontend: Next.js
standalone output.

## Conventions

- **Clean architecture per module:** `handler` (HTTP/DTO) → `service` (rules) →
  `repository` (interface) → `repository_gorm` (adapter). Services depend on
  interfaces, never concrete types. Service layer never imports Gin/GORM. `pkg/`
  never imports `internal/`. No import cycles. Cross-module calls go through
  exported ports, wired in `cmd/api/main.go`.
- **OpenAPI-first:** update `backend/api/openapi.yaml` in the same PR as a handler
  change; `docs/openapi.yaml` is the published snapshot. CI runs
  `openapi-spec-validator`. Swagger live at `/swagger`.
- **Migrations:** `backend/migrations/NNNNNN_name.{up,down}.sql` (golang-migrate),
  zero-padded strictly increasing. **Current head: `000014`; next is `000015`.**
  Every up has a matching down. Immutable once merged — fix forward. Seed is
  idempotent (upsert by natural key).
- **Config:** 12-factor via env, validated at startup. Backend vars: `PORT,
  DATABASE_URL, REDIS_URL, JWT_SECRET, ENV, LOG_LEVEL, CORS_ORIGINS,
  ACCESS_TOKEN_TTL, REFRESH_TOKEN_TTL, RESET_TOKEN_TTL, ANTHROPIC_API_KEY (opt),
  AI_MODEL, AI_ENABLED`. Frontend: `NEXT_PUBLIC_API_BASE_URL`.
- **API:** `/api/v1`, offset pagination (`page/page_size/sort/filter/q`), uniform
  error envelope (`pkg/apierror`), `Idempotency-Key` on retryable POSTs.
- **No TODOs** in merged code. Production-grade or not at all.
- **The `-p 1` rule:** integration/repository tests share one Postgres DB — run
  them serially (`go test ./... -p 1`) so packages don't race. Pure unit tests
  stay parallel.

## Endpoints the backend serves

`/api/v1/*` · `/healthz` (liveness) · `/readyz` (readiness: DB+Redis up +
migrations applied) · `/metrics` (Prometheus) · `/swagger`.

## 7-stage feature flow

Design → Database → Backend → API → Frontend → Tests → Documentation, in order,
no overlap. Definition of Done in `docs/07-ROADMAP.md` §1.3.

## Guardrails for agents

- Do **not** edit the numbered planning docs `docs/00..07` or `openapi.yaml`
  without explicit instruction — they are the design source of truth.
- Infra/CI/docs changes live in `infra/`, `.github/`, `docs/` (new files),
  `README.md`, `Makefile`, this file.
- Don't commit secrets. K8s `secret.yaml` and Helm `secrets.*` are placeholders;
  override at deploy time.
- Git: branch off `main`; don't commit/push unless asked.
