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

🚧 **In active development.** Phase 1 (planning & design) deliverables are complete
and live in [`docs/`](./docs). Implementation follows the milestone plan in the
[Development Roadmap](./docs/07-ROADMAP.md).

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
(optional) Kubernetes + Helm

## Monorepo layout

```
interviewos/
├── docs/            # PRD, SRS, architecture, schema, OpenAPI, design system, roadmap
│   └── diagrams/    # Mermaid source for ER / architecture diagrams
├── backend/         # Go API (clean architecture: cmd/{api,migrate,seed}, internal/<module>, pkg/, migrations/)
│   └── api/         # openapi.yaml — contract source of truth (maintained with the backend)
├── frontend/        # Next.js app (app/, features/, components/, lib/)
├── infra/           # docker-compose.yml, nginx, k8s, helm
├── Makefile         # dev/build/test targets (repo root)
└── .github/         # CI/CD workflows
```

## Getting started

> Local development setup lands with Milestone **M0 (Foundation)**. Once available:
>
> ```bash
> make dev        # boots postgres, redis, backend, frontend, nginx via infra/docker-compose.yml
> make migrate    # runs database migrations
> make seed       # loads DSA + System Design curriculum seed
> make test       # runs backend + frontend test suites
> ```
>
> See [Roadmap → M0](./docs/07-ROADMAP.md) for the exact bootstrap sequence.

## Development process

Built **feature by feature**. Every feature ships through all seven stages before
the next begins: **Design → Database → Backend → API → Frontend → Tests →
Documentation**. No TODOs; production-quality; clean architecture; SOLID.

## License

Proprietary — all rights reserved (pre-launch).
