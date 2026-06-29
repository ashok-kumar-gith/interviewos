# InterviewOS — Software Requirements Specification (SRS)

**Status:** Draft v1.0
**Owner:** Founding Engineering
**Last updated:** 2026-06-29
**Track at GA:** Backend SDE3
**Conforms to:** IEEE Std 830-1998 (recommended practice for SRS)
**Canonical source of intent:** [`01-PRD.md`](./01-PRD.md)

---

## Table of contents

1. [Introduction](#1-introduction)
2. [Overall description](#2-overall-description)
3. [Functional requirements](#3-functional-requirements)
4. [Non-functional requirements](#4-non-functional-requirements)
5. [External interface requirements](#5-external-interface-requirements)
6. [Spaced-repetition & readiness algorithm specification](#6-spaced-repetition--readiness-algorithm-specification)
7. [Traceability matrix](#7-traceability-matrix)
8. [Appendices](#8-appendices)

---

## 1. Introduction

### 1.1 Purpose

This Software Requirements Specification (SRS) defines the complete functional and
non-functional requirements for **InterviewOS**, a learning operating system for
software engineers preparing for technical interviews. It is the authoritative
engineering contract derived from the Product Requirements Document
([`01-PRD.md`](./01-PRD.md)). It is written for the founding engineering team
(backend, frontend, infra), QA, and downstream documents (ER diagram, database
schema, OpenAPI specification, test plans).

The SRS is scoped to the **General Availability (GA) release** of the **Backend
SDE3** track. Where the PRD marks a feature as post-GA (P2), this document
specifies it at a lower level of detail and flags it as out of GA scope.

### 1.2 Scope

InterviewOS is a single, opinionated web application whose north-star outcome is
that a user **never has to ask "what should I study today?"**. The system answers
that question automatically, every day, from a personalized curriculum, the
user's progress, confidence levels, a spaced-repetition revision engine, and the
user's target company and timeline.

**In scope (GA):**

- Authentication & accounts (Google OAuth, GitHub OAuth, email/password, JWT
  access + rotating refresh, password reset).
- Intake wizard and a **deterministic Curriculum Engine** that generates a dated,
  N-week (default 12) roadmap of plan days and tasks across six pillars.
- Dashboard & "Today" view, Master Roadmap (Week → Day → Task).
- Six content pillars: DSA, System Design (HLD), LLD, Backend Engineering,
  Behavioral, Resume.
- Design Problems catalog.
- Cross-cutting engines: Revision Engine (spaced repetition), Analytics Engine
  (readiness/streaks/weakness), Company Mode (roadmap re-weighting).
- Mock Interview module, Resource Library, Notifications (in-app), AI assistants
  (Claude-backed, optional with deterministic fallback).
- Production concerns: migrations, tests, CI/CD, observability, soft-delete.

**Out of scope (per PRD §3.2):** live human mock-interview marketplace; online
judge / code execution; native mobile apps; team/enterprise/billing/multi-tenant;
end-user content authoring CMS.

### 1.3 Definitions, acronyms, and abbreviations

| Term | Definition |
|------|------------|
| **Pillar** | One of the six preparation domains: DSA, System Design (HLD), LLD, Backend Engineering, Behavioral, Resume. |
| **Track** | A top-level preparation program (GA = Backend SDE3). Content is track-scoped. |
| **Topic / Subtopic** | A unit / sub-unit of learning within a pillar. |
| **Resource** | A global, deduplicated learning asset (book, video, article, GitHub repo, practice link). |
| **Problem** | A canonical DSA problem, merged across source lists, keyed by source problem. |
| **Pattern** | A DSA solving technique (e.g., sliding window, two pointers) linked M:N to problems. |
| **Design Problem** | A System Design (HLD) practice problem (e.g., URL Shortener, Uber). |
| **LLD Problem** | An object-oriented low-level design problem (e.g., Parking Lot). |
| **Roadmap** | A per-user, dated program: Roadmap → RoadmapWeek → PlanDay → PlanTask. |
| **PlanDay** | A single dated day in the roadmap holding a set of PlanTasks ("Today"). |
| **PlanTask** | A unit of work referencing a content item via polymorphic `(item_type, item_id)` plus a task `kind`. |
| **kind** | The action a PlanTask represents: `study`, `solve`, `read`, `watch`, `revise`, `mock`. |
| **RevisionItem** | Spaced-repetition state for a learned item (stage, interval, due date, last recall, active flag). |
| **Readiness** | A 0–100 explainable score, per-pillar and overall, of interview preparedness. |
| **Company Mode** | A mode that re-weights the roadmap by company-defined pillar/topic multipliers. |
| **Mock** | A recorded practice interview (coding / system design / LLD / behavioral / backend engineering) producing findings. |
| **STAR** | Situation, Task, Action, Result — the behavioral story structure. |
| **ATS** | Applicant Tracking System (resume keyword scanning). |
| **DSA / HLD / LLD** | Data Structures & Algorithms / High-Level Design / Low-Level Design. |
| **DDIA** | *Designing Data-Intensive Applications* (reference book). |
| **JWT** | JSON Web Token. |
| **OAuth** | Open Authorization (Google, GitHub providers). |
| **P0 / P1 / P2** | Priority: GA-blocking / GA-desirable / post-GA. |
| **FR / NFR** | Functional Requirement / Non-Functional Requirement. |
| **p95 / p99** | 95th / 99th percentile latency. |
| **WCAG AA** | Web Content Accessibility Guidelines, Level AA. |
| **SHALL** | Denotes a mandatory, testable requirement (RFC 2119 sense). |

### 1.4 References

| Ref | Document |
|-----|----------|
| R1 | InterviewOS PRD — [`01-PRD.md`](./01-PRD.md) |
| R2 | Database schema — `04-DATABASE-SCHEMA.md` (downstream) |
| R3 | OpenAPI / Swagger spec — `05-API-CONTRACTS.md` (downstream) |
| R4 | Release roadmap — `07-ROADMAP.md` (downstream) |
| R5 | IEEE Std 830-1998 — SRS recommended practice |
| R6 | RFC 7519 (JWT), RFC 6749 (OAuth 2.0), RFC 7636 (PKCE) |
| R7 | OWASP Application Security Verification Standard (ASVS) |
| R8 | WCAG 2.1 Level AA |

### 1.5 Document conventions

- Each requirement has a stable **ID** (`FR-<MODULE>-NNN`, `NFR-<CATEGORY>-NNN`).
- The keyword **SHALL** denotes a mandatory requirement. **SHOULD** denotes a
  strong recommendation. **MAY** denotes an optional behavior.
- **Priority** mirrors the PRD: **P0** (GA-blocking), **P1** (GA-desirable),
  **P2** (post-GA, specified for forward-compatibility).
- Every functional requirement lists **Preconditions** and **Acceptance criteria**;
  acceptance criteria are written to be directly translatable into automated tests.
- "User-data tables" means tables carrying a `deleted_at` soft-delete column per
  PRD §8.

---

## 2. Overall description

### 2.1 Product perspective

InterviewOS is a new, self-contained product (not a component of a larger system).
It is a client-server web application:

- **Frontend:** a responsive single-page-style React application that consumes a
  versioned REST API.
- **Backend:** a stateless Go API service backed by PostgreSQL (durable state) and
  Redis (cache, sessions/rate-limit counters, ephemeral engine state).
- **External services:** Google & GitHub OAuth (identity), the Claude API (AI
  assistants), and an email provider (post-GA, for password reset and
  notifications).

Curriculum content (tracks, pillars, topics, resources, problems, design problems,
companies, weights) is **curated and seeded via database migrations**, not authored
by end users. The Curriculum Engine and Revision Engine are **deterministic**; AI
only augments and must never block core flows.

```
┌──────────────┐     HTTPS / REST (/api/v1)     ┌─────────────────┐
│  Web client  │  ───────────────────────────▶  │   Go API (Gin)  │
│ React 19/Next│  ◀───────────────────────────  │   stateless     │
└──────────────┘                                 └────────┬────────┘
       │                                                  │
       │ OAuth redirect                          ┌────────┴─────────┐
       ▼                                         ▼                  ▼
┌──────────────┐                          ┌───────────┐     ┌────────────┐
│ Google/GitHub│                          │ PostgreSQL│     │   Redis    │
└──────────────┘                          └───────────┘     └────────────┘
                                                 ▲
                                          ┌──────┴───────┐
                                          │  Claude API  │ (AI, optional)
                                          └──────────────┘
```

### 2.2 Product functions (summary)

1. Authenticate users (OAuth + email/password), maintain sessions via rotating
   refresh tokens, support password reset.
2. Capture intake and **generate a personalized, dated, N-week roadmap**.
3. Produce a concrete **"Today" plan** automatically every day.
4. Present the Master Roadmap and per-task detail; track progress, confidence,
   time, and notes; support reschedule/skip.
5. Deliver six content pillars plus a Design Problems catalog, all over a unified,
   deduplicated content model.
6. Schedule and process **spaced-repetition revisions** (1/3/7/15/30 days).
7. Compute **readiness** (overall + per-pillar), streaks, weakest/strongest topics,
   and an estimated interview-readiness date.
8. Re-weight the roadmap for a **target company** (Company Mode).
9. Record **mock interviews** and generate remediation tasks from findings.
10. Surface **notifications** (in-app at GA).
11. Provide **AI assistants** with deterministic fallback.

### 2.3 User classes and characteristics

User classes reuse the PRD personas (PRD §4). GA persona focus is **Priya**.

| Class | Persona | Characteristics | Frequency of use |
|-------|---------|-----------------|------------------|
| **Senior preparer (primary)** | Priya — Senior backend eng (5 yrs) → SDE3 at FAANG | Strong coder; rusty system design; no behavioral prep. Wants structure, SD depth, readiness signal. | Daily |
| **Leveling-up preparer** | Arjun — SDE2 → SDE3 | Solid fundamentals; needs distributed-systems depth + LLD. | Daily |
| **Returning preparer** | Meena — returning after a break | Needs full-spectrum ramp incl. DSA refresh; adaptive pacing; revision discipline. | Daily |
| **Company targeter** | Has an Amazon loop in 6 weeks | Wants Company Mode, compressed timeline, LP/behavioral focus. | Daily, time-boxed |
| **System (engine) actors** | Curriculum/Revision/Analytics engines, schedulers | Non-human actors that generate plans, schedule revisions, snapshot readiness. | Continuous / scheduled |

All human users are individual end users; there are **no admin/org roles at GA**
(content is seeded). All users have identical capabilities once authenticated.

### 2.4 Operating environment

- **Client:** modern evergreen browsers — latest two stable versions of Chrome,
  Edge, Firefox, and Safari (desktop and mobile). Responsive web only; no native
  apps at GA. Minimum supported viewport width 360 px.
- **Server:** Linux containers (Docker), orchestrated via Docker Compose for GA,
  fronted by Nginx (TLS termination, reverse proxy, static asset serving).
- **Datastores:** PostgreSQL (primary, version ≥ 15) and Redis (version ≥ 7).
- **CI/CD:** GitHub Actions (lint, test, build, image publish).

### 2.5 Design and implementation constraints

The following stack is mandated and constrains design:

**Frontend**
- React **19** + **Next.js** (App Router) + **TypeScript** (strict mode).
- **Tailwind CSS** + **shadcn/ui** for the design system.
- **TanStack Table** for data grids; **Recharts** for charts.
- **React Query (TanStack Query)** for server-state; **Zustand** for client UI
  state; **React Hook Form (RHF)** for forms (with schema validation).

**Backend**
- **Go** + **Gin** HTTP framework.
- **PostgreSQL** via **GORM**; SQL migrations are versioned and forward/backward
  runnable.
- **Redis** for cache, rate-limit counters, refresh-token/session bookkeeping.
- **JWT** for access/refresh tokens.
- **Swagger / OpenAPI** generated and served for the API.

**Infrastructure**
- **Docker** + **Docker Compose**; **GitHub Actions** CI/CD; **Nginx** reverse proxy.

**Constraints derived from the stack & PRD decisions**
- The API is **versioned** under `/api/v1` and is the only contract between client
  and server.
- The Curriculum and Revision engines are **deterministic** and implemented
  server-side; identical inputs SHALL produce identical outputs (AI is additive).
- Polymorphic plan tasks use `(item_type, item_id)` + `kind` (PRD §8/§9).
- All user-data tables use **soft-delete** (`deleted_at`).
- Passwords are hashed with **bcrypt or argon2id** (never reversible).

### 2.6 Assumptions and dependencies

- **A1.** Curriculum content for at least DSA and System Design is seeded before
  GA; other pillars are seeded progressively per the release plan (R4).
- **A2.** Google and GitHub OAuth applications are registered and credentials are
  provisioned via environment configuration.
- **A3.** A Claude API key is available; if absent or the call fails, AI features
  degrade gracefully to deterministic fallbacks (PRD §7.18, §9).
- **A4.** Email delivery is **not** required at GA (in-app notifications and
  reset-token display/log path suffice); email is a post-GA dependency (PRD OQ2).
- **A5.** Revision intervals are **fixed** at `1/3/7/15/30` days at GA (PRD OQ1).
- **A6.** Default curriculum length is **12 weeks** and is configurable per user
  via intake.
- **A7.** A single user operates a single active track (Backend SDE3) at GA.
- **A8.** Server clock is authoritative for due-date computation; the engine uses
  the user's configured time zone for "day" boundaries.

---

## 3. Functional requirements

> Notation: each requirement is `ID | SHALL-statement | Priority | Preconditions |
> Acceptance criteria`. IDs are stable and referenced by the traceability matrix
> (§7).

### 3.1 Authentication & accounts (`FR-AUTH`) — PRD §7.1, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-AUTH-001** | The system SHALL allow a new user to sign up with email and password. | P0 |
| **FR-AUTH-002** | The system SHALL allow sign-up and sign-in via **Google OAuth 2.0**. | P0 |
| **FR-AUTH-003** | The system SHALL allow sign-up and sign-in via **GitHub OAuth 2.0**. | P0 |
| **FR-AUTH-004** | The system SHALL hash passwords using **bcrypt or argon2id** and SHALL never store or log plaintext passwords. | P0 |
| **FR-AUTH-005** | On successful authentication the system SHALL issue a short-lived **JWT access token** and a **rotating refresh token**. | P0 |
| **FR-AUTH-006** | The system SHALL **rotate** the refresh token on every refresh and invalidate the previously used refresh token. | P0 |
| **FR-AUTH-007** | The system SHALL detect refresh-token **reuse** of an already-rotated token and revoke the entire token family (session) on reuse. | P0 |
| **FR-AUTH-008** | The system SHALL provide a **forgot-password** flow that issues a single-use, time-limited password-reset token. | P0 |
| **FR-AUTH-009** | The system SHALL allow a user to **log out**, invalidating the active session/refresh token. | P0 |
| **FR-AUTH-010** | The system SHALL link multiple OAuth providers and an email/password credential to a **single User** when they share a verified email, recorded via `OAuthAccount`. | P1 |
| **FR-AUTH-011** | The system SHALL reject sign-up with an email that already belongs to an existing account and SHALL guide the user to sign in or link instead. | P0 |
| **FR-AUTH-012** | The system SHALL enforce a minimum password policy (≥ 8 characters; reject the top common-password denylist). | P1 |
| **FR-AUTH-013** | The system SHALL expire access tokens within **15 minutes** and refresh tokens within **30 days** (configurable). | P0 |

- **Preconditions (FR-AUTH-002/003):** OAuth provider apps configured (A2).
- **Preconditions (FR-AUTH-008):** an account with that email exists.
- **Acceptance criteria:**
  - A user can complete sign-up via each of the three methods and reach intake.
  - A session survives access-token expiry via refresh **without re-login**, and a
    rotated (old) refresh token is rejected.
  - Submitting a valid reset token with a new password lets the user sign in with
    the new password; reusing the same reset token a second time is rejected.
  - For an unknown email, the forgot-password endpoint returns a **generic success
    response** (no account enumeration).

### 3.2 Intake & Curriculum Engine (`FR-CUR`) — PRD §7.2, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-CUR-001** | The system SHALL present an **intake wizard** capturing: years of experience, target company, target role/level, available hours/week, preferred start date, and self-assessed strength (1–5) per pillar. | P0 |
| **FR-CUR-002** | The system SHALL persist intake answers to `UserProfile`. | P0 |
| **FR-CUR-003** | The system SHALL generate an ordered, **dated**, N-week `Roadmap` (default N = 12) of `RoadmapWeek` → `PlanDay` → `PlanTask` on intake completion. | P0 |
| **FR-CUR-004** | Generated `PlanTask`s SHALL each carry: linked content item `(item_type, item_id)`, `kind`, estimated hours, priority, and difficulty. | P0 |
| **FR-CUR-005** | The engine SHALL respect the user's **weekly hour budget**: total planned hours in any week SHALL NOT exceed the stated budget by more than **10%**. | P0 |
| **FR-CUR-006** | The engine SHALL **front-load lower self-assessed pillars** (allocate proportionally more time to weaker pillars). | P0 |
| **FR-CUR-007** | The engine SHALL apply the **target company's weights** (if a company is chosen) when ordering and allocating tasks (see `FR-CMP`). | P0 |
| **FR-CUR-008** | The engine SHALL be **deterministic**: the same intake inputs and content version SHALL produce an identical roadmap. | P0 |
| **FR-CUR-009** | The engine SHALL respect content **prerequisite ordering** (a topic's prerequisites are scheduled on or before that topic). | P0 |
| **FR-CUR-010** | The system SHALL allow the user to **regenerate** the roadmap after editing intake, preserving already-recorded progress where the content item still exists in the new plan. | P1 |
| **FR-CUR-011** | The engine SHALL ensure the **start-date PlanDay is non-empty** (at least one task). | P0 |
| **FR-CUR-012** | The engine SHALL distribute **revise** and **mock** task kinds into the plan only as produced by the Revision Engine (`FR-REV`) and Mock module (`FR-MOCK`), respectively; the initial generation SHALL contain study/solve/read/watch tasks. | P0 |
| **FR-CUR-013** | The system SHALL support an optional **AI refinement** pass over the deterministic roadmap that may reorder/annotate tasks but SHALL NOT be required for a valid roadmap (graceful fallback). | P1 |

- **Preconditions:** authenticated user; seeded content for the active track (A1).
- **Acceptance criteria:**
  - Submitting intake produces a full 12-week roadmap with a non-empty Today plan
    for the start date.
  - For every week, `sum(estimated_hours) ≤ 1.10 × weekly_budget`.
  - Two intakes with identical inputs and content version yield byte-identical
    roadmaps (excluding generated IDs/timestamps).
  - A topic never appears before any of its declared prerequisites.

### 3.3 Dashboard & "Today" view (`FR-DASH`) — PRD §7.3, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-DASH-001** | The dashboard SHALL display **overall readiness** and **per-pillar readiness** (0–100). | P0 |
| **FR-DASH-002** | The dashboard SHALL display the current **study streak** and the **estimated interview-readiness date**. | P0 |
| **FR-DASH-003** | The dashboard SHALL render a **"Today" task list** of the current PlanDay's tasks across all kinds (study/solve/read/watch/revise/mock). | P0 |
| **FR-DASH-004** | The system SHALL support **one-tap task completion** from the Today list. | P0 |
| **FR-DASH-005** | Completing a task SHALL update readiness, streak, and Today state **without a full page reload** (optimistic update reconciled with server). | P0 |
| **FR-DASH-006** | The dashboard SHALL render charts: **readiness over time** (line), **time-spent calendar heatmap**, and **per-pillar radar**. | P0 |
| **FR-DASH-007** | When the user has **fallen behind** (incomplete tasks from prior days), the Today view SHALL surface a clearly labeled **carry-over** section and offer one-action reschedule (see `FR-ROAD-006`). | P0 |
| **FR-DASH-008** | The "Today" view SHALL be derived from the PlanDay matching the **user's local date** (per A8). | P0 |
| **FR-DASH-009** | When no plan exists, the dashboard SHALL prompt the user to complete intake. | P0 |

- **Preconditions:** authenticated user with a generated roadmap (except FR-DASH-009).
- **Acceptance criteria:**
  - Dashboard renders all readiness metrics and a correct Today list (per NFR-PERF-001).
  - Completing a Today task updates the displayed readiness and streak within the
    same view without a navigation/reload.
  - On a day with overdue tasks, a carry-over section is shown with the overdue count.

### 3.4 Master Roadmap (`FR-ROAD`) — PRD §7.4, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-ROAD-001** | The system SHALL present a **Week → Day → Task** hierarchy for the user's roadmap. | P0 |
| **FR-ROAD-002** | Each `PlanTask` view SHALL expose: objectives, topic/subtopic, linked resources, linked problems, estimated hours, deliverables, priority, difficulty, progress status, confidence (1–5), notes, and attachments. | P0 |
| **FR-ROAD-003** | The system SHALL allow the user to **mark a task complete / in-progress / pending**. | P0 |
| **FR-ROAD-004** | The system SHALL allow the user to set a **confidence rating (1–5)** and free-text **notes** per task/topic/problem. | P0 |
| **FR-ROAD-005** | The system SHALL compute **day-level progress** as a function of its tasks' statuses. | P0 |
| **FR-ROAD-006** | The system SHALL allow the user to **reschedule** an incomplete task/day forward to a chosen future date, shifting it into that PlanDay. | P0 |
| **FR-ROAD-007** | The system SHALL allow the user to **skip** a task, marking it `skipped` (excluded from "behind" counts and from completion-based readiness, but retained for audit). | P0 |
| **FR-ROAD-008** | When a task is rescheduled, dependent **revision schedules** SHALL anchor to the actual completion date, not the originally planned date. | P0 |
| **FR-ROAD-009** | The system SHALL allow attaching **files/links** (`Attachment`) and **notes** (`Note`) polymorphically to a task, topic, or problem. | P1 |
| **FR-ROAD-010** | The system SHALL allow the user to **add an ad-hoc task** to a PlanDay (referencing existing content). | P1 |
| **FR-ROAD-011** | Marking a learning task **complete** SHALL create a corresponding `RevisionItem` (see `FR-REV-001`) when the task kind is `study`, `solve`, `read`, or `watch`. | P0 |

- **Preconditions:** authenticated user with a roadmap.
- **Acceptance criteria:**
  - A user can open any day, mark tasks complete, set confidence/notes, and
    reschedule incomplete tasks forward; the source day no longer lists the moved task.
  - Skipping a task removes it from the "behind" count and from coverage
    denominators but the action is recorded in `AuditLog`.
  - Completing a `study` task creates exactly one `RevisionItem` due **+1 day**.

### 3.5 DSA module (`FR-DSA`) — PRD §7.5, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-DSA-001** | The system SHALL present a **unified DSA curriculum** merging Blind 75, NeetCode 150, Grind 75, and Tech Interview Handbook into a **canonical problem set**, deduplicated by source problem. | P0 |
| **FR-DSA-002** | Each canonical `Problem` SHALL map to **≥ 1 `Pattern`** via `ProblemPattern`. | P0 |
| **FR-DSA-003** | Each `Problem` SHALL record its origins via `ProblemSource` (Blind75 / NeetCode / Grind75 / …) and a **canonical outbound link** (e.g., LeetCode). | P0 |
| **FR-DSA-004** | Each `Problem` SHALL carry **difficulty** and **per-company frequency** via `ProblemCompanyFrequency`. | P0 |
| **FR-DSA-005** | The system SHALL provide, per DSA topic: concept, pattern, visual explanation, problem list, revision schedule linkage, common mistakes, and expected questions. | P0 |
| **FR-DSA-006** | The system SHALL allow **filtering** problems by pattern, difficulty, and company; and **sorting** by company frequency. | P0 |
| **FR-DSA-007** | The system SHALL track per-problem progress (`UserProblemProgress`): status, confidence (1–5), time spent, notes, and attempt count. | P0 |
| **FR-DSA-008** | There SHALL be **no duplicate canonical problems**: a uniqueness constraint on the canonical key prevents duplicates across source lists. | P0 |

- **Preconditions:** DSA content seeded (A1).
- **Acceptance criteria:**
  - Counting canonical problems yields no duplicates by canonical key.
  - Every problem returns ≥ 1 pattern; filtering by pattern/difficulty/company
    returns only matching problems.
  - Marking a problem solved records confidence/time and creates a `RevisionItem`.

### 3.6 System Design module (`FR-SD`) — PRD §7.6, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-SD-001** | The system SHALL present a System Design curriculum synthesized from System Design Primer, DDIA, ByteByteGo/Alex Xu, Gaurav Sen, and engineering blogs. | P0 |
| **FR-SD-002** | Each SD topic SHALL expose: theory, videos, articles, examples, tradeoffs, practice designs, and interview & follow-up questions. | P0 |
| **FR-SD-003** | SD topics SHALL reference `Resource`s many-to-many via `TopicResource` (dedup; see `FR-LIB`). | P0 |
| **FR-SD-004** | The system SHALL track per-topic progress and confidence (`UserTopicProgress`). | P0 |

- **Preconditions:** SD content seeded (A1).
- **Acceptance criteria:** opening any SD topic shows all structured sections;
  completing a topic creates a revision item and updates SD pillar coverage.

### 3.7 Backend Engineering module (`FR-BE`) — PRD §7.7, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-BE-001** | The system SHALL present a Backend Engineering depth curriculum covering: Kafka, Redis, SQL, NoSQL, transactions, MVCC, indexes, replication, partitioning, CAP, consensus (Raft/Paxos/ZooKeeper), load balancing, Nginx, CDN, Docker, Kubernetes, networking, Linux, concurrency, memory, performance, profiling, and Go runtime (goroutines, channels, GC). | P0 |
| **FR-BE-002** | Each Backend topic SHALL expose theory + resources + expected interview questions and track per-topic progress/confidence. | P0 |
| **FR-BE-003** | Backend topics SHALL declare **prerequisites** where applicable, honored by the Curriculum Engine (`FR-CUR-009`). | P0 |

- **Acceptance criteria:** all listed subject areas exist as topics; completing a
  topic feeds revision and Backend pillar coverage.

### 3.8 LLD module (`FR-LLD`) — PRD §7.8, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-LLD-001** | The system SHALL present LLD theory: SOLID, design patterns (GoF), and UML. | P0 |
| **FR-LLD-002** | The system SHALL present OO design problems: Parking Lot, BookMyShow, Splitwise, Elevator, Chess, Food Delivery, Ride Sharing (`LLDProblem`). | P0 |
| **FR-LLD-003** | Each `LLDProblem` SHALL expose structured sections (requirements, entities/classes, relationships/UML, patterns applied, code/skeleton notes, edge cases). | P0 |
| **FR-LLD-004** | The system SHALL track per-LLD-problem progress and confidence. | P0 |

- **Acceptance criteria:** all seven OO problems are present with structured
  sections; completing one feeds revision and LLD pillar coverage.

### 3.9 Design Problems module (`FR-DESIGN`) — PRD §7.9, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-DESIGN-001** | The system SHALL present an **ordered catalog** of HLD design problems: URL Shortener, TinyURL, Pastebin, Notification System, WhatsApp, Slack, Instagram, Twitter, YouTube, Uber, Google Docs, Payment Gateway. | P0 |
| **FR-DESIGN-002** | Each `DesignProblem` SHALL expose: requirements, capacity estimation, API, schema, caching, queueing, tradeoffs, scaling, failure handling, alternative architectures, and interview tips. | P0 |
| **FR-DESIGN-003** | Design problems SHALL be ordered by increasing difficulty and SHALL respect that ordering in the roadmap. | P0 |
| **FR-DESIGN-004** | The system SHALL track per-design-problem progress and confidence. | P0 |

- **Acceptance criteria:** all listed design problems exist, ordered by difficulty;
  opening one shows all structured sections.

### 3.10 Behavioral module (`FR-BHV`) — PRD §7.10, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-BHV-001** | The system SHALL provide a **STAR story builder** capturing Situation, Task, Action, Result as discrete fields (`BehavioralStory`). | P0 |
| **FR-BHV-002** | The system SHALL support themes: leadership, ownership, conflict, failure, mentorship, stakeholder management, project rescue, production incidents, ambiguity, impact. | P0 |
| **FR-BHV-003** | A story SHALL be **taggable to ≥ 1 theme** and reusable across themes. | P0 |
| **FR-BHV-004** | The system SHALL provide an **AI Story Improver** suggesting stronger framing, metrics, and concision; it SHALL degrade gracefully when AI is disabled (returns no suggestions, never blocks save). | P1 |
| **FR-BHV-005** | The system SHALL track behavioral readiness as **theme coverage** (themes with ≥ 1 sufficiently-complete story). | P0 |

- **Acceptance criteria:** a user can create, tag, edit, and save a STAR story
  with all four fields; behavioral coverage increases as themes gain stories.

### 3.11 Resume module (`FR-RES`) — PRD §7.11, P1

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-RES-001** | The system SHALL provide a **project builder** (`ResumeProject`) capturing title, role, stack, and bullet points. | P1 |
| **FR-RES-002** | The system SHALL provide an **impact/metrics builder** prompting for quantified outcomes per project. | P1 |
| **FR-RES-003** | The system SHALL compute a **resume score** based on metric density, action-verb usage, and length heuristics. | P1 |
| **FR-RES-004** | The system SHALL perform an **ATS keyword check** against the target role/company and report missing keywords. | P1 |
| **FR-RES-005** | The system SHALL offer an **AI Resume Reviewer** with graceful fallback to deterministic scoring. | P1 |

- **Acceptance criteria:** building a project with metrics raises the resume score;
  ATS check lists missing keywords for the chosen target.

### 3.12 Mock Interview module (`FR-MOCK`) — PRD §7.12, P1

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-MOCK-001** | The system SHALL record a **`MockInterview`** of type coding / system design / LLD / behavioral / backend engineering with date, duration, and outcome score. | P1 |
| **FR-MOCK-002** | The system SHALL record structured **`MockFinding`s** (weakness areas mapped to topics/pillars). | P1 |
| **FR-MOCK-003** | Mock findings SHALL feed the **Weakness Detector** (Analytics) and SHALL generate **remediation `PlanTask`s** placed into upcoming PlanDays. | P1 |
| **FR-MOCK-004** | A scheduled mock SHALL appear in "Today" as a `mock`-kind task when due. | P1 |

- **Acceptance criteria:** recording a mock with a finding mapped to a topic
  creates a remediation task referencing that topic in a future PlanDay.

### 3.13 Revision Engine (`FR-REV`) — PRD §7.13, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-REV-001** | On completion of a learning item, the system SHALL create a `RevisionItem` with `interval = 1 day` and `due_at = completion_date + 1 day` (per the user's local day). | P0 |
| **FR-REV-002** | The system SHALL surface **due revision items** as `revise`-kind tasks in "Today" on or after `due_at`. | P0 |
| **FR-REV-003** | On a revision, the system SHALL record **recall accuracy** (correct / incorrect). | P0 |
| **FR-REV-004** | On **correct** recall, the system SHALL **advance** the interval to the next step in `[1, 3, 7, 15, 30]` and set the next `due_at` accordingly. | P0 |
| **FR-REV-005** | On **incorrect** recall, the system SHALL **reset** the interval to `1` day and set `due_at = today + 1 day`. | P0 |
| **FR-REV-006** | When an item is at the final interval (`30`) and recalled correctly, the system SHALL mark it **graduated** (`is_active=false`) and SHALL stop scheduling further revisions for it. | P0 |
| **FR-REV-007** | The system SHALL **dedupe** revision items: at most **one active (non-graduated) `RevisionItem` per (user, item_type, item_id)**; re-completing an item with an active revision SHALL NOT create a duplicate. | P0 |
| **FR-REV-008** | Overdue revision items SHALL remain due (roll forward) and SHALL count against revision health (`§6.2`) until completed or skipped. | P0 |
| **FR-REV-009** | A user MAY **skip** a due revision; a skipped revision SHALL be re-scheduled to `today + 1` and counted as a miss for revision-health purposes. | P1 |
| **FR-REV-010** | Revision intervals SHALL be **fixed** at `1/3/7/15/30` at GA (PRD OQ1). | P0 |

- **Preconditions:** the source learning item is completed (`FR-ROAD-011`).
- **Acceptance criteria:**
  - Completing a topic creates a revision task due **+1 day**.
  - Correct recall advances `1→3→7→15→30`; incorrect recall resets to `1`.
  - A correct recall at interval `30` graduates the item (no further revisions).
  - Re-completing an item with an active revision does not create a second item.

### 3.14 Company Mode (`FR-CMP`) — PRD §7.14, P0 (engine) / P1 (full set)

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-CMP-001** | The system SHALL model `Company` entities with **pillar/topic weight multipliers** via `CompanyWeight`. | P0 |
| **FR-CMP-002** | When a target company is selected, the engine SHALL apply its weights to **re-rank and re-allocate** roadmap tasks and to weight readiness emphasis. | P0 |
| **FR-CMP-003** | The GA company set SHALL include: Amazon, Google, Microsoft, Uber, Flipkart, Atlassian, Rubrik, PhonePe, Razorpay, Swiggy; **≥ 3 fully weighted** (Amazon, Google, Uber), the remainder scaffolded (PRD OQ3). | P1 |
| **FR-CMP-004** | The system SHALL surface **company-specific problem frequency** (via `ProblemCompanyFrequency`) when a company is targeted. | P0 |
| **FR-CMP-005** | Changing the target company SHALL allow roadmap **re-weighting** (regeneration per `FR-CUR-010`) while preserving recorded progress. | P0 |
| **FR-CMP-006** | Company weighting SHALL still respect the weekly hour budget (`FR-CUR-005`) and prerequisites (`FR-CUR-009`). | P0 |

- **Acceptance criteria:** selecting Amazon increases the relative allocation/ranking
  of behavioral (LP) and DSA tasks versus the no-company baseline, without exceeding
  the weekly budget by >10%.

### 3.15 Resource Library (`FR-LIB`) — PRD §7.15, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-LIB-001** | The system SHALL maintain a **global, deduplicated `Resource`** catalog with type (book/video/article/github/practice), estimated time, priority, and difficulty. | P0 |
| **FR-LIB-002** | Topics SHALL reference resources **many-to-many** via `TopicResource`. | P0 |
| **FR-LIB-003** | A resource SHALL be **deduplicated** by canonical URL/identity (no two resource rows with the same canonical identity). | P0 |
| **FR-LIB-004** | The system SHALL allow the user to **filter** the library by type, pillar, difficulty, and priority. | P1 |
| **FR-LIB-005** | The system SHALL track per-resource consumption status (e.g., read/watched) where the resource appears as a `read`/`watch` task. | P1 |

- **Acceptance criteria:** the same resource referenced by multiple topics resolves
  to a single `Resource` row; filtering returns only matching resources.

### 3.16 Analytics Engine (`FR-ANALYTICS`) — PRD §7.16, P0

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-ANALYTICS-001** | The system SHALL compute **completion %** per pillar and overall. | P0 |
| **FR-ANALYTICS-002** | The system SHALL compute **average confidence** per pillar and overall. | P0 |
| **FR-ANALYTICS-003** | The system SHALL compute **revision accuracy** and **revision health** (`§6.2`). | P0 |
| **FR-ANALYTICS-004** | The system SHALL identify the **weakest and strongest topics/pillars** (lowest/highest readiness contributors). | P0 |
| **FR-ANALYTICS-005** | The system SHALL track **time spent** (`StudySession`) per topic/pillar/day. | P0 |
| **FR-ANALYTICS-006** | The system SHALL compute and maintain the **study streak** (`StreakDay`): consecutive local days with ≥ 1 completed task. | P0 |
| **FR-ANALYTICS-007** | The system SHALL compute **per-pillar and overall readiness** per the model in `§6.2`. | P0 |
| **FR-ANALYTICS-008** | The system SHALL persist a **daily `ReadinessSnapshot`** to power the readiness-over-time chart. | P0 |
| **FR-ANALYTICS-009** | The system SHALL compute an **Estimated Interview Readiness Date** by projecting the recent readiness-gain rate to the target readiness threshold. | P0 |
| **FR-ANALYTICS-010** | The system SHALL incorporate **mock scores** into readiness/weakness when mocks exist. | P1 |
| **FR-ANALYTICS-011** | Readiness weights SHALL be **configurable** (server config), and the system SHALL be able to **explain** a readiness score by its component contributions. | P1 |

- **Acceptance criteria:** completing tasks raises completion % and readiness; a
  daily snapshot exists for each active day; the estimated readiness date is a
  future date when readiness < threshold and resolves to "ready" at/above threshold.

### 3.17 Notifications (`FR-NOTIF`) — PRD §7.17, P1

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-NOTIF-001** | The system SHALL generate **in-app notifications** for: today's tasks ready, revision due, weekly review, missed goals, and streak reminders (`Notification`). | P1 |
| **FR-NOTIF-002** | The user SHALL be able to **mark notifications read** and view unread counts. | P1 |
| **FR-NOTIF-003** | Notifications SHALL be **in-app at GA**; email/push channels are **post-GA** (P2). | P1 |
| **FR-NOTIF-004** | Notification generation SHALL be **idempotent** per (user, type, day): no duplicate notifications for the same trigger on the same day. | P1 |

- **Acceptance criteria:** when revisions become due, a single "revision due"
  in-app notification appears for that day; marking it read decrements the unread count.

### 3.18 AI features (`FR-AI`) — PRD §7.18, P1

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-AI-001** | The system SHALL provide AI assistants backed by the **Claude API**: Study Planner, Interview Coach, Resume Reviewer, Story Improver, Weakness Detector, Daily Planner, System Design Reviewer. | P1 |
| **FR-AI-002** | Every AI feature SHALL **degrade gracefully** to a deterministic fallback (or a clear "AI unavailable" state) when AI is disabled, mis-configured, rate-limited, or errors; AI SHALL NEVER block a core flow. | P1 |
| **FR-AI-003** | AI responses SHOULD be **cached** keyed on input to limit cost/latency. | P1 |
| **FR-AI-004** | AI features SHALL be **toggleable** (per-feature flag) and SHALL respect a per-user AI on/off preference. | P1 |
| **FR-AI-005** | AI prompts SHALL NOT include secrets or other users' data; only the requesting user's data SHALL be sent. | P0 |
| **FR-AI-006** | The Daily Planner / Study Planner AI SHALL only **refine** the deterministic plan (reorder/annotate), never be the source of truth (PRD §9). | P1 |

- **Acceptance criteria:** with AI disabled, all dependent features still function
  via fallback; with AI enabled, requests are cached and only the requesting user's
  data is sent.

### 3.19 Audit & soft-delete (`FR-AUDIT`) — PRD §8

| ID | Requirement | Priority |
|----|-------------|----------|
| **FR-AUDIT-001** | All **user-data tables** SHALL implement **soft-delete** via `deleted_at`; deletes SHALL set the timestamp and exclude rows from default queries. | P0 |
| **FR-AUDIT-002** | Soft-deleted records SHALL be **recoverable** by an internal restore path within a retention window (`NFR-DATA-002`). | P1 |
| **FR-AUDIT-003** | Security-relevant and mutating actions (auth, reschedule, skip, delete, regeneration) SHALL be recorded in `AuditLog`. | P0 |

- **Acceptance criteria:** a deleted entity disappears from default listings but is
  present (with `deleted_at`) in the table; the action appears in `AuditLog`.

---

## 4. Non-functional requirements

### 4.1 Performance (`NFR-PERF`)

| ID | Requirement | Target |
|----|-------------|--------|
| **NFR-PERF-001** | Dashboard initial render latency. | **p95 < 2 s** (PRD §10). |
| **NFR-PERF-002** | API read-endpoint latency. | **p99 < 300 ms** (PRD §10). |
| **NFR-PERF-003** | API write-endpoint latency (excluding AI). | p99 < 600 ms. |
| **NFR-PERF-004** | Roadmap generation for a 12-week plan. | < 3 s server-side (synchronous). |
| **NFR-PERF-005** | Task-completion round trip (optimistic UI). | UI reflects completion < 100 ms; server confirm < 300 ms. |
| **NFR-PERF-006** | AI-backed endpoints. | Async or streamed; SHALL NOT block non-AI flows; client shows progress state. |

### 4.2 Scalability (`NFR-SCALE`)

| ID | Requirement | Target |
|----|-------------|--------|
| **NFR-SCALE-001** | The API service SHALL be **stateless** and horizontally scalable behind Nginx. | N replicas, no sticky sessions (session state in Redis/DB). |
| **NFR-SCALE-002** | The system SHALL support at least **10,000 registered users / 1,000 concurrent active** at GA on commodity infra. | Sustained, within latency targets. |
| **NFR-SCALE-003** | Read-heavy content endpoints SHALL be **cacheable** (Redis / HTTP cache headers). | Content reads served from cache where possible. |

### 4.3 Security (`NFR-SEC`)

| ID | Requirement | Target |
|----|-------------|--------|
| **NFR-SEC-001** | Passwords SHALL be hashed with **bcrypt (cost ≥ 12) or argon2id**; never stored/logged in plaintext. | Verified by code review + tests. |
| **NFR-SEC-002** | Access tokens SHALL be short-lived JWTs (≤ 15 min); refresh tokens rotated and reuse-detected (`FR-AUTH-006/007`). | Enforced. |
| **NFR-SEC-003** | All traffic SHALL be over **TLS**; Nginx terminates TLS and sets HSTS. | TLS 1.2+ only. |
| **NFR-SEC-004** | The API SHALL enforce **rate limiting** (per-IP and per-user) on auth and mutating endpoints (Redis counters). | e.g., ≤ 10 auth attempts / min / IP. |
| **NFR-SEC-005** | The system SHALL mitigate the **OWASP Top 10** (injection via parameterized GORM queries, broken-auth controls, IDOR via per-user authorization checks, etc.). | Per OWASP ASVS Level 2. |
| **NFR-SEC-006** | Every data access SHALL enforce **per-user authorization**: a user SHALL only access their own progress/notes/roadmap/etc. | IDOR tests pass. |
| **NFR-SEC-007** | OAuth flows SHALL use **state** (CSRF) and **PKCE** where supported. | Enforced. |
| **NFR-SEC-008** | Secrets (JWT signing key, OAuth secrets, Claude key, DB creds) SHALL be loaded from **environment/secret store**, never committed. | CI secret-scan. |
| **NFR-SEC-009** | Password-reset and email-existence responses SHALL avoid **account enumeration** (generic responses). | Verified. |
| **NFR-SEC-010** | Security headers (CSP, X-Content-Type-Options, X-Frame-Options/Frame-Ancestors, Referrer-Policy) SHALL be set. | Set at Nginx/app. |

### 4.4 Reliability & availability (`NFR-REL`)

| ID | Requirement | Target |
|----|-------------|--------|
| **NFR-REL-001** | Crash-free sessions. | **≥ 99.5%** (PRD §10). |
| **NFR-REL-002** | API service availability. | ≥ 99.5% monthly. |
| **NFR-REL-003** | Engine determinism & idempotency: re-running generation/snapshot/revision processing SHALL NOT corrupt or duplicate state. | Idempotent; verified. |
| **NFR-REL-004** | Database migrations SHALL be **reversible** and run automatically in CI/CD before deploy. | Up/down tested. |
| **NFR-REL-005** | Failure of the AI provider, email provider, or cache SHALL **degrade gracefully**, not fail the core app. | Verified. |
| **NFR-REL-006** | Daily readiness snapshots and notification generation SHALL be **idempotent** if re-run. | No duplicates. |

### 4.5 Usability & accessibility (`NFR-USE`)

| ID | Requirement | Target |
|----|-------------|--------|
| **NFR-USE-001** | The UI SHALL support **dark mode** (and light), persisted per user. | PRD §7.3/G8. |
| **NFR-USE-002** | The UI SHALL be **responsive** down to 360 px width. | No horizontal body scroll. |
| **NFR-USE-003** | Core flows (complete task, navigate Today/Roadmap, mark revision) SHALL be **keyboard-driven** with discoverable shortcuts. | PRD G8. |
| **NFR-USE-004** | The UI SHALL meet **WCAG 2.1 Level AA**: contrast, focus visibility, labels/ARIA, keyboard operability. | Audited. |
| **NFR-USE-005** | All interactive controls SHALL have accessible names and visible focus states. | Audited. |
| **NFR-USE-006** | Loading and error states SHALL be explicit (skeletons, toasts), never silent. | Verified. |

### 4.6 Maintainability (`NFR-MAINT`)

| ID | Requirement | Target |
|----|-------------|--------|
| **NFR-MAINT-001** | Backend and frontend SHALL have automated **unit + integration tests** in CI. | Engines (Curriculum, Revision, Analytics) ≥ 85% line coverage. |
| **NFR-MAINT-002** | Code SHALL pass **lint/format/type checks** in CI (Go vet/golangci-lint; TS strict + ESLint). | CI gate. |
| **NFR-MAINT-003** | The data model SHALL be **track-scoped** to allow new tracks **without schema/engine rewrites** (PRD §9). | Design review. |
| **NFR-MAINT-004** | The API SHALL be **versioned** (`/api/v1`); breaking changes require a new version. | Enforced. |
| **NFR-MAINT-005** | Configuration (intervals, readiness weights, budgets) SHALL be externalized, not hard-coded inline. | Config module. |

### 4.7 Observability (`NFR-OBS`)

| ID | Requirement | Target |
|----|-------------|--------|
| **NFR-OBS-001** | The backend SHALL emit **structured (JSON) logs** with request ID, user ID (where applicable), and latency. | Correlatable. |
| **NFR-OBS-002** | The backend SHALL expose **metrics** (request rate, error rate, latency histograms, engine timings, cache hit rate). | Scrapeable (e.g., Prometheus format). |
| **NFR-OBS-003** | The system SHALL support **distributed tracing** with a propagated correlation/trace ID across request → DB/Redis/Claude spans. | Trace IDs in logs. |
| **NFR-OBS-004** | Health and readiness **probes** SHALL be exposed (`/healthz`, `/readyz`). | 200 when healthy. |
| **NFR-OBS-005** | Errors SHALL never leak secrets/stack traces to clients; full detail goes to logs only. | Verified. |

### 4.8 Data retention & soft-delete (`NFR-DATA`)

| ID | Requirement | Target |
|----|-------------|--------|
| **NFR-DATA-001** | User-data tables SHALL use **soft-delete** (`deleted_at`); default queries exclude soft-deleted rows. | `FR-AUDIT-001`. |
| **NFR-DATA-002** | Soft-deleted user data SHALL be retained (recoverable) for **≥ 30 days** before any hard purge. | Retention policy. |
| **NFR-DATA-003** | A user SHALL be able to **export** their data and **request account deletion** (soft-delete → purge after window). | Privacy. |
| **NFR-DATA-004** | PII SHALL be limited to what intake requires; logs SHALL avoid PII beyond user ID. | Verified. |

### 4.9 Internationalization readiness (`NFR-I18N`)

| ID | Requirement | Target |
|----|-------------|--------|
| **NFR-I18N-001** | UI copy SHALL be **externalized** (no hard-coded user-facing strings) to enable future localization. | Strings in resource files. |
| **NFR-I18N-002** | Dates/times SHALL be stored in **UTC** and rendered in the **user's time zone**; day boundaries use the user's TZ (A8). | Verified. |
| **NFR-I18N-003** | The UI SHALL use Unicode/UTF-8 end-to-end and handle non-ASCII input. | Verified. |
| **NFR-I18N-004** | The default GA locale is **en-US**; no translated locales are required at GA. | Scope note. |

---

## 5. External interface requirements

### 5.1 User interfaces

- Responsive web app (React 19 / Next.js, Tailwind + shadcn/ui). Primary surfaces:
  Auth, Intake wizard, Dashboard/Today, Master Roadmap, per-pillar module views,
  Design Problems, Behavioral, Resume, Mock, Resource Library, Analytics,
  Notifications. Dark mode and keyboard shortcuts per `NFR-USE`.

### 5.2 Software interfaces — REST API

- **API-001.** The backend SHALL expose a **REST API versioned under `/api/v1`**,
  JSON over HTTPS, authenticated by Bearer JWT (except auth/public endpoints).
- **API-002.** The API SHALL publish a **Swagger / OpenAPI** document describing
  all endpoints, schemas, and error formats (R3).
- **API-003.** Errors SHALL use a **consistent envelope** (`code`, `message`,
  `details?`) and correct HTTP status codes.
- **API-004.** List endpoints SHALL support **pagination, filtering, and sorting**
  where applicable (e.g., DSA problems by pattern/difficulty/company).
- **API-005.** Mutating endpoints SHALL accept an **idempotency key** where retries
  are expected (e.g., task completion).

### 5.3 OAuth providers

- **EXT-OAUTH-001.** Integrate **Google OAuth 2.0** and **GitHub OAuth 2.0** for
  sign-in/sign-up, using state (CSRF) and PKCE where supported (`NFR-SEC-007`).
- **EXT-OAUTH-002.** Map provider identities to `OAuthAccount` linked to a `User`
  (account linking per `FR-AUTH-010`).

### 5.4 Claude API (AI)

- **EXT-AI-001.** AI features SHALL call the **Claude API** server-side; the key is
  never exposed to the client.
- **EXT-AI-002.** Calls SHALL have **timeouts, retries with backoff, and a circuit
  breaker**; on failure they fall back deterministically (`FR-AI-002`).
- **EXT-AI-003.** Responses SHOULD be cached (Redis) keyed on normalized input
  (`FR-AI-003`).

### 5.5 Email provider (post-GA)

- **EXT-EMAIL-001.** Password-reset and notification email is **post-GA (P2)**. At
  GA, reset tokens are delivered via an internal/no-email path; the interface is
  abstracted so an email provider can be plugged in without code changes elsewhere.

### 5.6 Datastore interfaces

- **EXT-DB-001.** **PostgreSQL** (≥ 15) via GORM is the system of record; access is
  via parameterized queries/migrations only.
- **EXT-REDIS-001.** **Redis** (≥ 7) provides caching, rate-limit counters, and
  refresh-token/session bookkeeping. Redis is **non-authoritative**: its loss
  degrades performance but not correctness.

### 5.7 Communication interfaces

- **COMM-001.** All client-server traffic over **HTTPS/TLS 1.2+**.
- **COMM-002.** CORS SHALL be restricted to the configured web origin(s).

---

## 6. Spaced-repetition & readiness algorithm specification

This section is the **normative** specification of the Revision Engine intervals
and the readiness scoring model. It refines `FR-REV` and `FR-ANALYTICS`.

### 6.1 Spaced-repetition (Revision Engine)

**Interval ladder (fixed at GA, PRD OQ1):**

```
INTERVALS = [1, 3, 7, 15, 30]   // days
```

**State per `RevisionItem`:** `stage` (0..4), `interval_days`,
`due_at` (UTC date), `last_recall` (`correct` | `incorrect` | null),
`is_active` boolean (graduated ⇒ `is_active=false`), `ease` (reserved / inert at GA;
stored, never mutated), `review_count`, `lapse_count`,
identity `(user_id, item_type, item_id)`.

**Creation (on learning-item completion — `FR-REV-001`):**

```
on complete(item):
    if exists active RevisionItem(user, item.type, item.id):   # dedup (FR-REV-007)
        return                                                 # no duplicate
    create RevisionItem:
        stage          = 0
        interval_days  = INTERVALS[0]            # = 1
        due_at         = local_day(completion_date) + 1 day
        is_active      = true
        last_recall    = null
```

**On revision (recall recorded — `FR-REV-003..006`):**

```
on review(revisionItem, recall):
    if recall == correct:
        if revisionItem.stage == len(INTERVALS) - 1:            # at 30
            revisionItem.is_active = false                      # graduated (FR-REV-006)
            revisionItem.last_recall = correct
            # no further due_at; stop scheduling
        else:
            revisionItem.stage          += 1
            revisionItem.interval_days   = INTERVALS[stage]
            revisionItem.due_at          = today_local + interval_days
            revisionItem.last_recall     = correct
    else:   # incorrect (FR-REV-005)
        revisionItem.stage          = 0
        revisionItem.interval_days  = INTERVALS[0]              # reset to 1
        revisionItem.due_at         = today_local + 1 day
        revisionItem.last_recall    = incorrect
```

**Due & overdue (`FR-REV-002/008`):** an `active` item is **due** when
`due_at <= today_local`. Overdue items remain due (roll forward) and count as
outstanding for revision health until reviewed or skipped. **Skip (`FR-REV-009`):**
sets `due_at = today_local + 1`, counted as a miss.

**Worked example:** complete topic on day D ⇒ due D+1. Correct on D+1 ⇒ due D+4
(interval 3). Correct ⇒ +7, +15, +30. Incorrect at any step ⇒ due tomorrow at
interval 1. Correct at interval 30 ⇒ graduated.

### 6.2 Readiness scoring model

Readiness is **explainable, tunable, and bounded to [0, 100]** (PRD §9). It is
computed **per pillar** and aggregated into **Overall Readiness**.

#### 6.2.1 Per-item inputs (within a pillar `p`)

For each learnable item `i` in pillar `p` (topic / problem / design / LLD / theme):

- **Coverage** `cov_i ∈ {0, 1}`: `1` if the item is **completed** (not just planned;
  skipped items are excluded from both numerator and denominator).
- **Confidence** `conf_i ∈ [0, 1]`: the user's 1–5 confidence normalized as
  `(rating − 1) / 4` (so 1★ → 0.0, 5★ → 1.0); if unrated but completed, default 0.5.
- **Revision health** `rev_i ∈ [0, 1]` from the item's active `RevisionItem`:
  - graduated (`is_active=false`) ⇒ `1.0`
  - active, current ⇒ `min(1.0, stage / 4)` blended with last recall:
    `rev_i = 0.5·(stage/4) + 0.5·(last_recall == correct ? 1 : 0)`
  - active, **overdue** ⇒ apply a decay `× max(0.3, 1 − 0.1·days_overdue)`
  - no revision item yet (just completed) ⇒ `0.5`

#### 6.2.2 Per-pillar readiness

Let `N_p` be the count of in-scope items for pillar `p` (excluding skipped). Define
pillar **coverage ratio**, **mean confidence over completed items**, and **mean
revision health over completed items**:

```
coverage_p   = (Σ cov_i) / N_p                       # 0..1   (breadth)
confidence_p = (Σ_{cov_i=1} conf_i) / max(1, Σ cov_i) # 0..1   (depth)
revhealth_p  = (Σ_{cov_i=1} rev_i)  / max(1, Σ cov_i) # 0..1   (retention)
```

Per-pillar readiness combines breadth with depth and retention:

```
readiness_p = 100 × coverage_p × ( w_conf · confidence_p + w_rev · revhealth_p )
              where  w_conf + w_rev = 1     (defaults: w_conf = 0.6, w_rev = 0.4)
```

Rationale: a pillar cannot be "ready" without **breadth** (coverage gates the
score), and breadth alone is insufficient — it is scaled by **confidence** and
**revision health**. Mocks, when present, blend in (`FR-ANALYTICS-010`):

```
readiness_p ← (1 − w_mock)·readiness_p + w_mock·(100·mock_score_p)   # w_mock default 0 until a mock exists, else 0.2
```

#### 6.2.3 Overall readiness (weighted by pillar)

Let `W_p` be the **pillar weight** — the company weight (`CompanyWeight`,
normalized) when a company is targeted, else uniform across the six pillars:

```
Overall = Σ_p ( W_p × readiness_p )     with  Σ_p W_p = 1
```

All weights (`w_conf`, `w_rev`, `w_mock`, `W_p`) SHALL be **server-configurable**
(`NFR-MAINT-005`, `FR-ANALYTICS-011`), and the system SHALL be able to **explain**
a score by exposing `coverage_p`, `confidence_p`, `revhealth_p`, and `W_p`.

#### 6.2.4 Estimated Interview Readiness Date (`FR-ANALYTICS-009`)

Let `R_today` be current overall readiness, `R_target` the target threshold
(default **80**), and `rate` the mean daily readiness gain over a trailing window
(default 14 days, from `ReadinessSnapshot`s):

```
if R_today >= R_target:        date = today  ("interview-ready")
elif rate <= 0:                date = undefined  (show "insufficient momentum")
else:                          days_remaining = ceil((R_target − R_today) / rate)
                               date = today + days_remaining
```

The estimate SHALL be recomputed at each daily snapshot and SHALL never be presented
as a guarantee.

#### 6.2.5 Streak (`FR-ANALYTICS-006`)

A **StreakDay** is any user-local day with ≥ 1 completed (non-skipped) task. The
**current streak** is the count of consecutive local days ending today (or
yesterday, if today has no completion yet) each having a StreakDay. A local day with
zero completions **breaks** the streak.

---

## 7. Traceability matrix

Each PRD §7 feature maps to one or more functional requirement groups; key NFRs and
the algorithm spec are cross-referenced.

| PRD feature (§7) | Priority | SRS requirements | Supporting NFR / spec |
|------------------|----------|------------------|------------------------|
| 7.1 Authentication & accounts | P0 | FR-AUTH-001…013 | NFR-SEC-001/002/004/007/009, EXT-OAUTH |
| 7.2 Intake & Curriculum Engine | P0 | FR-CUR-001…013 | NFR-PERF-004, NFR-REL-003, §6.2 |
| 7.3 Dashboard & "Today" | P0 | FR-DASH-001…009 | NFR-PERF-001/005, NFR-USE-001…006 |
| 7.4 Master Roadmap | P0 | FR-ROAD-001…011 | FR-REV-001 (revision link), FR-AUDIT-003 |
| 7.5 DSA module | P0 | FR-DSA-001…008 | FR-LIB, §6.1 |
| 7.6 System Design module | P0 | FR-SD-001…004 | FR-LIB-002 |
| 7.7 Backend Engineering module | P0 | FR-BE-001…003 | FR-CUR-009 |
| 7.8 LLD module | P0 | FR-LLD-001…004 | FR-REV |
| 7.9 Design Problems module | P0 | FR-DESIGN-001…004 | FR-CUR-009 |
| 7.10 Behavioral module | P0 | FR-BHV-001…005 | FR-AI-002 |
| 7.11 Resume module | P1 | FR-RES-001…005 | FR-AI-002 |
| 7.12 Mock Interview module | P1 | FR-MOCK-001…004 | FR-ANALYTICS-010 |
| 7.13 Revision Engine | P0 | FR-REV-001…010 | §6.1 (normative algorithm) |
| 7.14 Company Mode | P0/P1 | FR-CMP-001…006 | FR-CUR-007, §6.2.3 |
| 7.15 Resource Library | P0 | FR-LIB-001…005 | — |
| 7.16 Analytics | P0 | FR-ANALYTICS-001…011 | §6.2 (readiness model) |
| 7.17 Notifications | P1 | FR-NOTIF-001…004 | EXT-EMAIL-001 (post-GA) |
| 7.18 AI features | P1 | FR-AI-001…006 | EXT-AI-001…003, NFR-REL-005 |
| §8 Soft-delete / audit | P0 | FR-AUDIT-001…003 | NFR-DATA-001…004 |

---

## 8. Appendices

### 8.1 Priority legend

- **P0** — GA-blocking. Must ship for General Availability of the Backend SDE3 track.
- **P1** — GA-desirable. Targeted for GA; degrades gracefully if deferred.
- **P2** — Post-GA. Specified here only for forward compatibility (email/push
  channels, native apps, online judge, team/enterprise, user-authored content).

### 8.2 Assumed defaults (from PRD open questions §13)

- Revision intervals fixed at `1/3/7/15/30` (OQ1).
- Notifications in-app at GA; email post-GA (OQ2).
- Company set: engine + 3 fully weighted (Amazon/Google/Uber), rest scaffolded (OQ3).
- Curriculum length default 12 weeks; readiness target threshold default 80.

### 8.3 Requirement count

- **Functional requirements:** 128
  (AUTH 13, CUR 13, DASH 9, ROAD 11, DSA 8, SD 4, BE 3, LLD 4, DESIGN 4, BHV 5,
  RES 5, MOCK 4, REV 10, CMP 6, LIB 5, ANALYTICS 11, NOTIF 4, AI 6, AUDIT 3).
- **Non-functional requirements:** 49
  (PERF 6, SCALE 3, SEC 10, REL 6, USE 6, MAINT 5, OBS 5, DATA 4, I18N 4 — 9 categories).
- **External-interface requirements:** 16 (API 5, OAUTH 2, AI 3, EMAIL 1, DB/REDIS 2, COMM 2, plus UI narrative).

**Total numbered FR + NFR: 177.**

---

*End of SRS v1.0. This document is consistent with `01-PRD.md` and is the
authoritative input to the database schema (`04-DATABASE-SCHEMA.md`) and API spec
(`05-API-CONTRACTS.md`).*
