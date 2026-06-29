# InterviewOS — Product Requirements Document (PRD)

**Status:** Draft v1.0
**Owner:** Product / Founding Engineering
**Last updated:** 2026-06-29
**Track at GA:** Backend SDE3

---

## 1. Summary

InterviewOS is a learning operating system for software engineers preparing for
technical interviews, spanning the seniority spectrum from SDE1 to Staff Engineer.
It replaces the fragmented toolchain engineers cobble together today — Excel
trackers, Notion templates, todo apps, LeetCode lists, and scattered bookmarks —
with a single, opinionated application that **tells the user exactly what to do
every day** until they receive an offer.

The product's north star is this: a user should **never** have to ask
*"what should I study today?"* The system answers that question automatically,
every day, based on a personalized curriculum, the user's progress, their
confidence levels, a spaced-repetition revision engine, and their target company
and timeline.

The first supported track at general availability (GA) is **Backend SDE3**. The
architecture is explicitly designed to be extensible to additional tracks
(Frontend, Mobile, ML, EM, SDE1/2, Staff) without schema or engine rewrites.

---

## 2. Problem statement

Interview preparation for senior backend roles is a 10–16 week, multi-disciplinary
effort spanning Data Structures & Algorithms (DSA), System Design, Low-Level
Design (LLD), Backend Engineering depth, Behavioral storytelling, and Resume
craft. Today engineers manage this with:

- **Excel/Sheets trackers** — manual, no scheduling intelligence, no revision.
- **Notion templates** — beautiful but static; no analytics, no adaptivity.
- **Todo apps (Todoist)** — no domain knowledge, no curriculum.
- **LeetCode + curated lists (Blind 75, NeetCode, Grind 75)** — great problems,
  zero integration with the rest of prep, heavy duplication across lists.
- **Books/blogs (DDIA, System Design Primer, ByteByteGo)** — unstructured; no
  path, no tracking, no revision.

The result: engineers spend cognitive energy on *planning and bookkeeping* instead
of *learning*, lose track of what to revise, under-prepare weak areas, and walk
into interviews without a calibrated sense of readiness.

### Pain points (prioritized)

| # | Pain | Impact |
|---|------|--------|
| P1 | "What do I study today?" decision fatigue | Daily friction, procrastination |
| P2 | No revision discipline — things learned are forgotten | Knowledge decay |
| P3 | No unified curriculum — duplication across DSA lists, scattered SD resources | Wasted time |
| P4 | No readiness signal — "am I ready?" is a guess | Wrong-timed applications |
| P5 | No adaptivity to target company | Generic, inefficient prep |
| P6 | Behavioral & resume prep are afterthoughts | Failed loops despite strong coding |

---

## 3. Goals & non-goals

### 3.1 Goals (GA — Backend SDE3 track)

- **G1.** Generate a personalized N-week curriculum (default 12) from a short intake.
- **G2.** Produce a concrete, ordered **daily plan** every day, automatically.
- **G3.** Unify DSA, System Design, LLD, Backend Engineering, Behavioral, and
  Resume into one curriculum with deduplicated resources.
- **G4.** Implement a spaced-repetition **revision engine** (1/3/7/15/30-day).
- **G5.** Track confidence, completion, and time-spent per topic/problem.
- **G6.** Compute an **Estimated Interview Readiness Date** and per-pillar readiness.
- **G7.** Adapt the roadmap to a target **company** (topic emphasis weighting).
- **G8.** Provide premium-quality UX: dark mode, responsive, fast, keyboard-driven.
- **G9.** Offer AI assistants (planner, coach, resume/story reviewer, weakness detector).
- **G10.** Be production-grade: auth, migrations, tests, CI/CD, docs, observability.

### 3.2 Non-goals (explicitly out of scope for GA)

- **N1.** Live human-to-human mock interview marketplace / scheduling with peers.
- **N2.** Executing/judging code submissions in a sandbox (no online judge at GA;
  problems link out to LeetCode and track status/confidence locally).
- **N3.** Mobile native apps (responsive web only at GA).
- **N4.** Team/enterprise/cohort features, billing, and multi-tenant orgs (post-GA).
- **N5.** Content authoring CMS for end users (curriculum is curated/seeded).

---

## 4. Target users & personas

| Persona | Description | Primary need |
|---------|-------------|--------------|
| **Priya — Senior backend eng (5 yrs), targeting SDE3 at FAANG** | Strong coder, rusty on system design, no behavioral prep | Structured 12-week plan, SD depth, readiness signal |
| **Arjun — SDE2 leveling up to SDE3** | Solid fundamentals, needs distributed-systems depth + LLD | Backend engineering module, design problems |
| **Meena — returning after a break** | Needs full-spectrum ramp incl. DSA refresh | Adaptive pacing, revision engine |
| **Company-targeter** | Has an Amazon loop in 6 weeks | Company Mode, compressed timeline, LP/behavioral focus |

GA persona focus: **Priya** (Senior → SDE3, FAANG/top-tier).

---

## 5. Product pillars

InterviewOS is organized around **six preparation pillars**. Readiness is computed
per-pillar and aggregated into Overall Readiness.

1. **DSA** — patterns, problems, company frequency, revision.
2. **System Design (HLD)** — theory + ordered design problems (URL shortener → Uber).
3. **LLD** — SOLID, design patterns, UML, OO design problems.
4. **Backend Engineering** — Kafka, Redis, SQL/NoSQL, transactions, consensus, Go runtime, etc.
5. **Behavioral** — STAR story builder across leadership/conflict/failure/ownership themes.
6. **Resume** — project/impact/metrics builder, ATS scoring.

Cross-cutting engines that operate over all pillars:

- **Curriculum Engine** (generates roadmap & daily plans)
- **Revision Engine** (spaced repetition)
- **Analytics Engine** (readiness, streaks, weak/strong topics)
- **Company Mode** (re-weights the plan)
- **Mock Interview module** (records results, feeds weakness detection)
- **AI Assistants** (planner, coach, reviewers)

---

## 6. Core user journey

```
Sign up (Google / GitHub / Email)
        ↓
Intake wizard (experience, target company, target role, hours/week, start date)
        ↓
Curriculum Engine generates a 12-week personalized roadmap
        ↓
┌──────────────────────────────────────────────────────────┐
│  EVERY DAY:                                                │
│  Dashboard → "Today" view auto-generated:                  │
│    • Study these topics      • Solve these problems        │
│    • Read this chapter       • Watch this video            │
│    • Revise these notes      • Give this mock (when due)   │
│  User completes tasks → logs confidence + time + notes     │
│        ↓                                                   │
│  Revision Engine schedules future revisions                │
│  Analytics updates readiness + streak                      │
│  Roadmap adapts if user falls behind / changes target      │
└──────────────────────────────────────────────────────────┘
        ↓
Readiness reaches threshold → "You're interview-ready" → apply → offer
```

---

## 7. Feature requirements

Each feature is specified with priority (P0 = GA-blocking, P1 = GA-desirable,
P2 = post-GA) and acceptance criteria. Full functional detail lives in the SRS
(`02-SRS.md`); this section captures product intent.

### 7.1 Authentication & accounts — **P0**
- Google OAuth, GitHub OAuth, and email/password login.
- JWT access tokens (short-lived) + refresh tokens (rotating).
- Forgot-password flow with email reset tokens.
- **Acceptance:** A user can sign up via any of the three methods, stay logged in
  across sessions via refresh, and reset a forgotten password.

### 7.2 Intake & Curriculum Engine — **P0**
- Intake captures: years of experience, target company, target role/level,
  available hours/week, preferred start date, self-assessed pillar strengths.
- Engine generates an ordered, dated, N-week roadmap of **plan days**, each with
  tasks across pillars, estimated hours, priority, and difficulty.
- Engine respects weekly hour budget and company weighting.
- **Acceptance:** Submitting intake produces a full 12-week roadmap with a
  non-empty "Today" plan for the start date, and total weekly hours never exceed
  the user's stated budget by more than 10%.

### 7.3 Dashboard & "Today" view — **P0**
- Overall + per-pillar readiness, study streak, estimated readiness date.
- "Today" task list (study/solve/read/watch/revise/mock) with one-tap completion.
- Charts: readiness over time, time-spent heatmap (calendar), pillar radar.
- **Acceptance:** Dashboard renders all readiness metrics and a correct Today list
  in < 2s p95; completing a task updates metrics without full reload.

### 7.4 Master Roadmap — **P0**
- Week → Day → Task hierarchy. Each task carries: objectives, topic/subtopic,
  resources, problems, estimated hours, deliverables, priority, difficulty,
  progress status, confidence (1–5), notes, attachments.
- Day-level and task-level progress tracking; reschedule/skip support.
- **Acceptance:** User can view any day, mark tasks complete, set confidence/notes,
  and reschedule incomplete tasks forward.

### 7.5 DSA module — **P0**
- Unified curriculum merging Blind 75, NeetCode 150, Grind 75, Tech Interview
  Handbook, deduplicated to a canonical problem set keyed by source problem.
- Per topic: concept, pattern, visual explanation, problems (with difficulty &
  company frequency), revision schedule, common mistakes, expected questions.
- **Acceptance:** No duplicate canonical problems; every problem maps to ≥1 pattern;
  filtering by pattern/difficulty/company works.

### 7.6 System Design module — **P0**
- Curriculum synthesized from System Design Primer, DDIA, ByteByteGo/Alex Xu,
  Gaurav Sen, and engineering blogs (Netflix, Uber, Cloudflare, Google, Amazon).
- Per topic: theory, videos, articles, examples, tradeoffs, practice designs,
  interview & follow-up questions.

### 7.7 Backend Engineering module — **P0**
- Depth curriculum: Kafka, Redis, SQL, NoSQL, transactions, MVCC, indexes,
  replication, partitioning, CAP, consensus (Raft/Paxos/ZooKeeper), load
  balancing, Nginx, CDN, Docker, Kubernetes, networking, Linux, concurrency,
  memory, performance, profiling, Go runtime (goroutines, channels, GC).

### 7.8 LLD module — **P0**
- SOLID, design patterns, UML; OO design problems: Parking Lot, BookMyShow,
  Splitwise, Elevator, Chess, Food Delivery, Ride Sharing.

### 7.9 Design Problems module — **P0**
- Ordered catalog: URL Shortener, TinyURL, Pastebin, Notification System,
  WhatsApp, Slack, Instagram, Twitter, YouTube, Uber, Google Docs, Payment Gateway.
- Each: requirements, capacity estimation, API, schema, caching, queueing,
  tradeoffs, scaling, failure handling, alternative architectures, interview tips.

### 7.10 Behavioral module — **P0**
- STAR story builder across themes: leadership, ownership, conflict, failure,
  mentorship, stakeholder management, project rescue, production incidents,
  ambiguity, impact.
- AI story improver suggests stronger framing, metrics, and concision.

### 7.11 Resume module — **P1**
- Project builder, impact/metrics builder, resume scoring, ATS keyword check.

### 7.12 Mock Interview module — **P1**
- Record results for coding / system design / LLD / behavioral / backend
  engineering mocks; weakness detection feeds the analytics engine and generates
  remediation tasks.

### 7.13 Revision Engine — **P0**
- Spaced repetition with intervals 1/3/7/15/30 days. On completing a learning
  item, schedules a revision item; on revision, records recall accuracy and
  advances or resets the interval.
- **Acceptance:** Completing a topic creates a revision task due +1 day; correct
  recall advances to +3, +7, etc.; incorrect recall resets to +1.

### 7.14 Company Mode — **P0 (engine), P1 (full company set)**
- Companies: Amazon, Google, Microsoft, Uber, Flipkart, Atlassian, Rubrik,
  PhonePe, Razorpay, Swiggy. Each defines pillar/topic weight multipliers that
  re-rank the roadmap (e.g., Amazon emphasizes LP/behavioral + DSA).

### 7.15 Resource Library — **P0**
- Global, deduplicated resources (books/videos/articles/github/practice) with
  type, estimated time, priority, difficulty. Topics reference resources many-to-many.

### 7.16 Analytics — **P0**
- Completion %, average confidence, weakest/strongest topics, revision accuracy,
  time spent, study streak, mock scores, company readiness, predicted offer-readiness date.

### 7.17 Notifications — **P1**
- Today's tasks, revision due, weekly review, missed goals, streak reminders.
  In-app at GA; email/push post-GA.

### 7.18 AI features — **P1**
- AI Study Planner, Interview Coach, Resume Reviewer, Story Improver, Weakness
  Detector, Daily Planner, System Design Reviewer. Backed by the Claude API.
  All AI features degrade gracefully (deterministic fallback) when disabled.

---

## 8. Canonical domain model (overview)

This is the shared vocabulary that the SRS, ER diagram, schema, and OpenAPI spec
must conform to. Detailed columns/indexes live in `04-DATABASE-SCHEMA.md`.

**Identity & config**
- `User`, `OAuthAccount`, `Session`/`RefreshToken`, `UserProfile` (intake),
  `PasswordResetToken`.

**Content / curriculum library (track-scoped, seeded)**
- `Track` (e.g., Backend SDE3) → `Pillar` (DSA, SD, LLD, …) → `Topic` → `Subtopic`.
- `Resource` (global) ↔ `Topic` via `TopicResource` (M:N, dedup).
- `Problem` (DSA, canonical; merged from sources) ↔ `Pattern` via `ProblemPattern`.
  `ProblemSource` records origin (Blind75/NeetCode/Grind75/…).
- `DesignProblem` (HLD) and `LLDProblem` with structured sections.
- `Company` ↔ `Topic`/`Pillar` via `CompanyWeight`; `ProblemCompanyFrequency`.

**User progress / engine state**
- `Roadmap` (per user) → `RoadmapWeek` → `PlanDay` → `PlanTask`.
- `PlanTask` references a content item (topic/problem/resource/design/etc.) via a
  polymorphic `(item_type, item_id)` plus task `kind` (study/solve/read/watch/revise/mock).
- `UserTopicProgress`, `UserProblemProgress` (status, confidence, time, notes).
- `RevisionItem` (spaced-repetition state: interval, due_at, ease, last_recall).
- `Note`, `Attachment` (polymorphic, attachable to tasks/topics/problems).
- `BehavioralStory` (STAR fields), `ResumeProfile`/`ResumeProject`,
  `MockInterview` (+ `MockFinding`).
- `StudySession` (time tracking), `StreakDay`, `ReadinessSnapshot` (daily metrics).
- `Notification`, `AuditLog`. Soft-delete (`deleted_at`) on user-data tables.

---

## 9. Key product decisions & rationale

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Online judge? | **No** at GA; link out to LeetCode, track locally | Judge infra is a product in itself; not core to the planning value prop |
| Curriculum source | **Curated/seeded**, versioned via migrations | Quality & consistency; avoids UGC moderation |
| Daily plan generation | **Deterministic engine** + optional AI refinement | Reliability first; AI augments, never blocks |
| AI provider | **Claude API** | Best-in-class reasoning; aligns with stack |
| Polymorphic plan tasks | `(item_type,item_id)` + `kind` | One unified Today list across heterogeneous content |
| Multi-track | Track-scoped content from day 1 | Extensibility without rewrite |
| Readiness model | Weighted per-pillar coverage × confidence × revision health | Explainable, tunable |

---

## 10. Success metrics

**Activation**
- % of signups that complete intake and generate a roadmap (target ≥ 70%).
- % that complete ≥1 task on day 1 (target ≥ 50%).

**Engagement / retention**
- D7 / D30 retention (targets 40% / 20%).
- Median study streak length; weekly active days.
- % of due revision tasks completed on time (target ≥ 60%).

**Outcome**
- Self-reported readiness lift (pre/post pillar self-assessment).
- % users reporting an offer (qualitative survey).

**Quality / reliability**
- Dashboard p95 < 2s; API p99 < 300ms for read endpoints.
- Crash-free sessions ≥ 99.5%.

---

## 11. Release plan (high level)

Detailed milestones, sequencing, and a feature-by-feature build order are in
`07-ROADMAP.md`. Summary:

- **M0 Foundation:** repo, CI/CD, auth, DB migrations, design system, app shell.
- **M1 Curriculum core:** content model + seed (DSA/SD), intake, Curriculum Engine,
  Roadmap + Today view, progress tracking.
- **M2 Engines:** Revision Engine, Analytics/readiness, Company Mode.
- **M3 Depth modules:** Backend Engineering, LLD, Design Problems, Behavioral.
- **M4 Polish & AI:** Resume, Mock, AI assistants, notifications, analytics charts.
- **M5 Hardening:** test coverage, perf, observability, docs, deploy.

---

## 12. Risks & mitigations

| Risk | Mitigation |
|------|-----------|
| Content curation is large & slow | Seed DSA+SD first (M1); structured seed format; expand per milestone |
| AI cost/latency/unreliability | Deterministic engine is source of truth; AI optional & cached; graceful fallback |
| Scope creep (12 modules) | Strict P0/P1/P2; ship Backend SDE3 vertical fully before breadth |
| Readiness model mis-calibration | Make weights configurable; expose explanation; iterate from cohort data |
| Copyright on curated content | Link to sources; store our own summaries/metadata, not copyrighted text |

---

## 13. Open questions

- OQ1: Should revision intervals be user-tunable or fixed at GA? *(Default: fixed 1/3/7/15/30.)*
- OQ2: Do we need email delivery at GA or is in-app sufficient? *(Default: in-app GA, email M4+.)*
- OQ3: Company set at GA — all 10 or top 3 (Amazon/Google/Uber)? *(Default: engine + 3 fully weighted, rest scaffolded.)*

These are deferred to confirmation; defaults are assumed for downstream documents.
