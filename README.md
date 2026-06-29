# InterviewOS

> The learning operating system for software engineers preparing for technical
> interviews — from SDE1 to Staff Engineer. First track at GA: **Backend SDE3**.

InterviewOS replaces the fragmented prep toolchain (Excel trackers, Notion
templates, todo apps, LeetCode lists, scattered bookmarks) with **one
application** that tells you exactly what to study every day until you get an
offer. Think *Notion × Linear × LeetCode × ByteByteGo × Todoist × an LMS*.

You should never have to ask *"what should I study today?"* — InterviewOS answers
it for you, every day, based on a personalized curriculum, your progress and
confidence, a spaced-repetition revision engine, and your target company and
timeline.

---

## Status

🚧 **In active development — hardening for GA.** The full planning set lives in
[`docs/`](./docs) and the product is built through the milestone plan: **M0
(Foundation + Auth), M1 (Curriculum core), M2 (Engines), M3 (Depth modules), and
M4 (Polish & AI) are implemented; M5 (hardening, infra, docs, deploy) is in
progress.** See the [Development Roadmap](./docs/07-ROADMAP.md).

The backend is a Go modular monolith (Gin + GORM) with 20+ domain modules,
migrations `000001`–`000014`, and a contract-first OpenAPI spec; the frontend is
a Next.js 15 / React 19 SPA. Both run locally via Docker (`make dev`) **or** a
sudo-free `~/.local` toolchain (`scripts/dev-local.sh`) on machines without Docker.

### Live endpoints (backend)

| Endpoint | Purpose |
|----------|---------|
| `/api/v1/*` | REST API (auth, intake, roadmap, today, pillars, revision, analytics, …) |
| `/healthz` | liveness (process up) |
| `/readyz` | readiness (Postgres + Redis reachable, migrations applied) |
| `/metrics` | Prometheus metrics |
| `/swagger` | live Swagger UI |

### Operational & contributor docs

| Doc | Purpose |
|-----|---------|
| [SETUP.md](./docs/SETUP.md) | local dev — Docker (`make dev`) **and** sudo-free `~/.local` path; env vars; migrate/seed; tests |
| [DEPLOYMENT.md](./docs/DEPLOYMENT.md) | docker-compose prod, Kubernetes (kubectl + Kustomize), Helm, secrets, health/metrics/Swagger |
| [DEVELOPER.md](./docs/DEVELOPER.md) | architecture, module layout, the 7-stage feature flow, testing strategy (incl. the `-p 1` note), OpenAPI-first |
| [CLAUDE.md](./CLAUDE.md) | concise contributor/agent guide: commands, conventions, where things live |

## Planning & design documents

The foundational documents are the source of truth for the build. Read them in order:

| # | Document | Purpose |
|---|----------|---------|
| 01 | [Product Requirements (PRD)](./docs/01-PRD.md) | Vision, pillars, features, decisions, success metrics |
| 02 | [Software Requirements (SRS)](./docs/02-SRS.md) | Functional & non-functional requirements, algorithms |
| 03 | [System Architecture](./docs/03-ARCHITECTURE.md) | Clean architecture, engines, infra, diagrams |
| 04 | [Database Schema](./docs/04-DATABASE-SCHEMA.md) | Normalized Postgres schema, ER diagram, indexing |
| 05 | [API Contracts](./docs/05-API-CONTRACTS.md) + [`openapi.yaml`](./docs/openapi.yaml) | REST API conventions & OpenAPI 3.1 spec (published snapshot; see note below) |
| 06 | [Frontend Design System](./docs/06-DESIGN-SYSTEM.md) | Tokens, components, layout, motion, a11y |
| 07 | [Development Roadmap](./docs/07-ROADMAP.md) | Milestones M0–M5, build order, sprints |

**OpenAPI spec — two roles.** The source of truth is
[`backend/api/openapi.yaml`](./backend/api/openapi.yaml) — contract-first, generated
and maintained alongside the backend (`swag` annotations).
[`docs/openapi.yaml`](./docs/openapi.yaml) is the committed, published snapshot of
that contract, used for planning and rendered docs.

## Product pillars

1. **DSA** — patterns, problems, company frequency, revision
2. **System Design (HLD)** — theory + ordered design problems
3. **LLD** — SOLID, design patterns, UML, OO problems
4. **Backend Engineering** — Kafka, Redis, SQL/NoSQL, consensus, Go runtime, …
5. **Behavioral** — STAR story builder
6. **Resume** — project / impact / metrics builder + ATS scoring

Cross-cutting engines: **Curriculum**, **Revision** (spaced repetition),
**Analytics** (readiness), **Company Mode**, **Mock Interviews**, **AI assistants**.

## Tech stack

**Frontend:** React 19 · Next.js (App Router) · TypeScript · TailwindCSS ·
shadcn/ui · TanStack Table · React Query · Zustand · React Hook Form · Recharts

**Backend:** Go · Gin · PostgreSQL · Redis · GORM · JWT · Swagger/OpenAPI

**Infra:** Docker · Docker Compose · GitHub Actions · Makefile · Nginx ·
Kubernetes (Kustomize) + Helm (`infra/k8s/`, `infra/helm/interviewos/`)

## Monorepo layout

```
interviewos/
├── docs/            # PRD, SRS, architecture, schema, OpenAPI, design system, roadmap
│   └── diagrams/    # Mermaid source for ER / architecture diagrams
├── backend/         # Go API (clean architecture: cmd/{api,migrate,seed}, internal/<module>, pkg/, migrations/)
│   └── api/         # openapi.yaml — contract source of truth (maintained with the backend)
├── frontend/        # Next.js app (app/, features/, components/, lib/)
├── infra/           # docker-compose.yml, nginx/, k8s/ (kustomize), helm/interviewos/
├── scripts/         # dev-local.sh — sudo-free local stack (no Docker)
├── Makefile         # dev/build/test/migrate/seed/k8s/helm targets (repo root)
├── CLAUDE.md        # contributor / agent quick reference
└── .github/         # CI/CD workflows
```

## Getting started

**With Docker:**

```bash
make dev        # boots postgres, redis, mailhog, backend, frontend, nginx via infra/docker-compose.yml
make migrate    # applies database migrations (cmd/migrate up)
make seed       # loads DSA + System Design curriculum seed (idempotent)
make be-test    # backend unit tests   (make be-test-integration for serial -p 1 tests)
make fe-build   # frontend build
```

**Without Docker (sudo-free `~/.local` toolchain):**

```bash
scripts/dev-local.sh start-db    # postgres (:5433) + redis (:6379) from ~/.local
scripts/dev-local.sh migrate     # apply migrations
scripts/dev-local.sh seed        # load seed
scripts/dev-local.sh backend     # run the Go API (:8080)
scripts/dev-local.sh frontend    # run the Next.js dev server (:3000)
```

See [SETUP.md](./docs/SETUP.md) for the full bootstrap, env vars, and both paths;
[DEPLOYMENT.md](./docs/DEPLOYMENT.md) for Kubernetes/Helm/compose deploys.

## Development process

Built **feature by feature**. Every feature ships through all seven stages before
the next begins: **Design → Database → Backend → API → Frontend → Tests →
Documentation**. No TODOs; production-quality; clean architecture; SOLID.

## License

Proprietary — all rights reserved (pre-launch).
