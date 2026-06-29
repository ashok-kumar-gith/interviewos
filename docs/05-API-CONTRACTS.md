# InterviewOS — API Contracts (05)

**Status:** v1.0 (GA — Backend SDE3)
**Base URL:** `/api/v1` · **Stack:** Go (REST) · PostgreSQL + GORM · JWT auth
**Machine-readable spec:** [`openapi.yaml`](./openapi.yaml) (OpenAPI 3.1, validated)

This companion explains conventions and gives per-module endpoint tables and
representative examples. Field names and enum values match
[`04-DATABASE-SCHEMA.md`](./04-DATABASE-SCHEMA.md) exactly.

---

## 1. Conventions

### 1.1 Versioning
- All endpoints are under `/api/v1`. Breaking changes ship under a new prefix
  (`/api/v2`); additive changes are made in place.
- The current API version is also returned in the `X-API-Version` response header.

### 1.2 Authentication
- **Access token:** short-lived JWT (15 min) sent as `Authorization: Bearer <token>`.
- **Refresh token:** rotating, long-lived (30 days), delivered as an `HttpOnly`,
  `Secure`, `SameSite=Strict` cookie (`refresh_token`) and also accepted in the
  `/auth/refresh` body. Each refresh rotates the token within a `family_id`;
  reuse of a revoked token revokes the whole family (theft detection).
- OAuth (`google`, `github`) completes via `/auth/oauth/{provider}/callback`,
  returning the same `AuthTokensResponse` envelope as email/password.
- `security: [{ bearerAuth: [] }]` is the global default; only the auth bootstrap
  endpoints (`register`, `login`, `oauth callback`, `forgot/reset password`) and
  `refresh` (which uses `refreshAuth`) are exempt.

### 1.3 Pagination, filtering, sorting, search
List endpoints accept these query params:

| Param | Type | Default | Meaning |
|-------|------|---------|---------|
| `page` | int ≥ 1 | 1 | 1-based page number |
| `page_size` | int 1–100 | 20 | items per page |
| `sort` | string | resource default | comma-separated fields; `-` prefix = descending (e.g. `-frequency_score,title`) |
| `filter` | string | — | RHS-colon expression list, e.g. `difficulty:hard,priority:high` |
| `q` | string | — | full-text search (Postgres tsvector / trigram) |

Typed filters also exist as first-class query params where useful
(`difficulty`, `company_id`, `pattern_id`, `type`, `theme`, `status`, …).

Paginated responses use a consistent wrapper:

```json
{
  "data": [ /* items */ ],
  "meta": { "page": 1, "page_size": 20, "total": 137, "total_pages": 7 }
}
```

### 1.4 Error envelope
All non-2xx responses use:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "hours_per_week must be between 1 and 80",
    "request_id": "req_01HX...",
    "details": [ { "field": "hours_per_week", "message": "must be <= 80" } ]
  }
}
```

`details` is present for validation errors. `request_id` is echoed for support.

### 1.5 Standard error codes

| HTTP | `code` | When |
|------|--------|------|
| 400 | `BAD_REQUEST` | Malformed body/params |
| 401 | `UNAUTHENTICATED` | Missing/invalid/expired access token |
| 401 | `INVALID_CREDENTIALS` | Bad email/password |
| 401 | `REFRESH_TOKEN_INVALID` | Reused/revoked/expired refresh token |
| 403 | `FORBIDDEN` | Authenticated but not permitted |
| 404 | `NOT_FOUND` | Resource missing or not owned |
| 409 | `CONFLICT` | Duplicate (email) / active roadmap already exists |
| 422 | `VALIDATION_ERROR` | Field-level validation failed |
| 429 | `RATE_LIMITED` | Rate limit exceeded (see `Retry-After`) |
| 503 | `AI_UNAVAILABLE` | AI provider down; deterministic fallback may be returned with `used_fallback:true` |
| 500 | `INTERNAL` | Unexpected server error |

### 1.6 Rate limiting
- Default: **120 req/min** per user (authenticated) / **20 req/min** per IP for
  auth endpoints. AI endpoints: **10 req/min** per user.
- Responses include `X-RateLimit-Limit`, `X-RateLimit-Remaining`,
  `X-RateLimit-Reset`. On 429, `Retry-After` (seconds) is returned.

### 1.7 Idempotency
- Mutating operations that can be retried — `POST /roadmaps/generate`,
  `POST /tasks/{id}/complete`, AI POSTs — accept an `Idempotency-Key` header.
  The server stores the first result keyed by `(user, route, key)` (see
  `ai_invocations.idempotency_key`) and replays it for duplicate keys within the
  retention window, preventing double roadmap generation or double-counted time.

### 1.8 Timestamps & types
- All timestamps are RFC 3339 / ISO 8601 UTC (`...Z`). Dates are `YYYY-MM-DD`.
- IDs are UUID v4 strings. `confidence` is an integer 1–5. Time is in **minutes**.

---

## 2. Endpoint reference (per module)

Auth column: **P** = public (no token), **B** = bearer access token, **R** = refresh token.

### Auth
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | P | Register with email/password |
| POST | `/auth/login` | P | Login with email/password |
| GET | `/auth/oauth/{provider}/callback` | P | Google/GitHub OAuth callback |
| POST | `/auth/refresh` | R | Rotate refresh, issue new access token |
| POST | `/auth/logout` | B | Revoke current refresh-token family |
| POST | `/auth/forgot-password` | P | Request reset email |
| POST | `/auth/reset-password` | P | Reset password with token |
| GET | `/auth/me` | B | Current user |

### Profile / Intake
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/profile` | B | Get intake profile |
| PUT | `/profile` | B | Create/update intake profile |

### Curriculum
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/roadmaps/generate` | B | Generate roadmap from intake |
| GET | `/roadmaps/active` | B | Active roadmap (weeks summary) |
| GET | `/roadmaps/{roadmapId}/weeks/{weekNumber}` | B | Week with days |
| GET | `/plan-days/{date}` | B | Plan day for a date |
| GET | `/today` | B | Today's auto-generated plan |
| POST | `/tasks/{taskId}/complete` | B | Complete task (confidence+time+notes) |
| POST | `/tasks/{taskId}/skip` | B | Skip task |
| POST | `/tasks/{taskId}/reschedule` | B | Reschedule task to another date |

### Content
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/tracks` | B | List tracks |
| GET | `/pillars` | B | List pillars (filter by track) |
| GET | `/topics` | B | Browse topics (page/filter/sort/search) |
| GET | `/topics/{topicId}` | B | Topic with subtopics + resources |
| GET | `/resources` | B | Browse resources |

### DSA
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/problems` | B | List problems (company/difficulty/pattern/source) |
| GET | `/problems/{problemId}` | B | Problem + patterns/sources/frequency/progress |
| GET | `/patterns` | B | List DSA patterns |

### System Design
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/design-problems` | B | Ordered HLD catalog |
| GET | `/design-problems/{designProblemId}` | B | Design problem with all sections |

### Backend Engineering
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/backend-engineering/topics` | B | Backend-engineering depth topics |

### LLD
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/lld-problems` | B | List LLD problems |
| GET | `/lld-problems/{lldProblemId}` | B | LLD problem with all sections |

### Behavioral
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/behavioral-stories` | B | List my stories (filter by theme) |
| POST | `/behavioral-stories` | B | Create story |
| GET | `/behavioral-stories/{storyId}` | B | Get story |
| PUT | `/behavioral-stories/{storyId}` | B | Update story |
| DELETE | `/behavioral-stories/{storyId}` | B | Delete story |
| POST | `/behavioral-stories/{storyId}/improve` | B | AI-improve story |

### Resume
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/resume/profile` | B | Get resume profile |
| PUT | `/resume/profile` | B | Create/update resume profile |
| GET | `/resume/projects` | B | List resume projects |
| POST | `/resume/projects` | B | Add project |
| PUT | `/resume/projects/{projectId}` | B | Update project |
| DELETE | `/resume/projects/{projectId}` | B | Delete project |
| POST | `/resume/score` | B | ATS + AI score |

### Mock
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/mock-interviews` | B | List mocks (filter by type) |
| POST | `/mock-interviews` | B | Create mock |
| GET | `/mock-interviews/{mockId}` | B | Mock with findings |
| PUT | `/mock-interviews/{mockId}` | B | Update mock |
| DELETE | `/mock-interviews/{mockId}` | B | Delete mock |
| POST | `/mock-interviews/{mockId}/findings` | B | Add finding |

### Revision
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/revision/due` | B | Due revision items (today/overdue) |
| POST | `/revision/{revisionItemId}/recall` | B | Submit recall; advance/reset interval |

### Company
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/companies` | B | List companies |
| GET | `/companies/{companyId}` | B | Company with weights |
| GET | `/company/target` | B | Get my target company |
| PUT | `/company/target` | B | Set target company (re-weights roadmap) |

> **Note — `/company/target` is intentionally singular.** It is a "my target"
> singleton sub-resource (the caller's single target company), so it uses the
> singular `company` and exposes only GET/PUT — it is not a pluralization bug and
> not a typo for `/companies/target`.

### Analytics
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/dashboard` | B | Dashboard aggregate (overall + per-pillar readiness, streak, today summary, revision-due count) — the p95 < 2s endpoint |
| GET | `/analytics/readiness` | B | Overall + per-pillar readiness, ready date |
| GET | `/analytics/snapshots` | B | Readiness snapshots over time |
| GET | `/analytics/streak` | B | Study streak + per-day activity |
| GET | `/analytics/topics` | B | Weak/strong topics |
| GET | `/analytics/time-spent` | B | Time-spent aggregation (heatmap) |

### Notifications
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/notifications` | B | List notifications |
| POST | `/notifications/{notificationId}/read` | B | Mark one read |
| POST | `/notifications/read-all` | B | Mark all read |

### AI
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/ai/planner` | B | Study planner (refines roadmap) |
| POST | `/ai/coach` | B | Interview coach Q&A |
| POST | `/ai/resume-review` | B | Resume review |
| POST | `/ai/story-improve` | B | Story improver (by id or inline STAR) |
| POST | `/ai/weakness-detect` | B | Weakness detector |
| POST | `/ai/daily-plan` | B | Daily-plan refinement |
| POST | `/ai/sd-review` | B | System-design answer review |

All AI endpoints degrade gracefully: on provider failure they return a
deterministic fallback with `"used_fallback": true` (200) or `AI_UNAVAILABLE`
(503) when no fallback is meaningful.

### Scope notes (deferred / GA surface)

- **Notes & Attachments REST API is P1 / deferred (FR-ROAD-009).** The
  polymorphic `/notes` and `/attachments` endpoints are **not** part of the GA
  API surface. At GA, per-task notes are captured via the `notes` field on
  `POST /tasks/{taskId}/complete`. The `owner_type` enum is **reserved** for the
  future Notes/Attachments API.
- **`study_sessions` have no standalone endpoint at GA.** They are derived
  server-side from task completions (`POST /tasks/{taskId}/complete`); there is no
  session create/list/start/stop endpoint.

---

## 3. Representative examples

### 3.1 Login

**Request**
```http
POST /api/v1/auth/login
Content-Type: application/json

{ "email": "priya@example.com", "password": "correct-horse-battery" }
```

**Response `200`**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "rt_8f3c...e91",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": "8c4a1e2b-7d4a-4b1e-9f2a-2b6c1d0e3f4a",
    "email": "priya@example.com",
    "email_verified": true,
    "full_name": "Priya Sharma",
    "role": "user",
    "status": "active",
    "last_login_at": "2026-06-29T08:40:00Z",
    "created_at": "2026-05-01T10:00:00Z",
    "updated_at": "2026-06-29T08:40:00Z"
  }
}
```
The `refresh_token` is also set as an `HttpOnly` cookie.

### 3.2 Generate roadmap

**Request**
```http
POST /api/v1/roadmaps/generate
Authorization: Bearer eyJhbGci...
Idempotency-Key: 2f1c9a44-roadmap-001
Content-Type: application/json

{ "regenerate": false, "use_ai": false }
```

**Response `201`** (weeks abbreviated)
```json
{
  "id": "b1d2c3e4-0000-4a1b-9c2d-aa11bb22cc33",
  "user_id": "8c4a1e2b-7d4a-4b1e-9f2a-2b6c1d0e3f4a",
  "track_id": "11111111-1111-4111-8111-111111111111",
  "profile_id": "22222222-2222-4222-8222-222222222222",
  "target_company_id": "aaaa1111-2222-4333-8444-555566667777",
  "start_date": "2026-07-01",
  "end_date": "2026-09-22",
  "total_weeks": 12,
  "hours_per_week": 15,
  "status": "active",
  "is_active": true,
  "generated_by": "engine",
  "weeks": [
    {
      "id": "w1-...","roadmap_id": "b1d2c3e4-...","week_number": 1,
      "start_date": "2026-07-01","end_date": "2026-07-07",
      "theme": "Arrays & Hashing + System Design intro",
      "focus_pillars": ["dsa","system_design"],
      "planned_hours": 15
    }
  ],
  "created_at": "2026-06-29T08:41:00Z",
  "updated_at": "2026-06-29T08:41:00Z"
}
```
A second call with the same `Idempotency-Key` replays this exact response. Calling
without `regenerate:true` while an active roadmap exists returns `409 CONFLICT`.

### 3.3 Get today

**Request**
```http
GET /api/v1/today
Authorization: Bearer eyJhbGci...
```

**Response `200`**
```json
{
  "id": "pd-2026-07-01",
  "roadmap_week_id": "w1-...",
  "user_id": "8c4a1e2b-7d4a-4b1e-9f2a-2b6c1d0e3f4a",
  "date": "2026-07-01",
  "planned_minutes": 150,
  "completed_minutes": 0,
  "is_rest_day": false,
  "summary": "Arrays & Hashing fundamentals + first SD read",
  "tasks": [
    {
      "id": "t-001",
      "plan_day_id": "pd-2026-07-01",
      "kind": "study",
      "item_type": "topic",
      "item_id": "topic-arrays-hashing",
      "pillar_type": "dsa",
      "title": "Study: Arrays & Hashing",
      "objectives": ["Hash map patterns", "Frequency counting"],
      "estimated_minutes": 60,
      "priority": "high",
      "difficulty": "easy",
      "status": "pending",
      "sort_order": 0,
      "confidence": null,
      "time_spent_minutes": null,
      "completed_at": null
    },
    {
      "id": "t-002",
      "plan_day_id": "pd-2026-07-01",
      "kind": "solve",
      "item_type": "problem",
      "item_id": "problem-two-sum",
      "pillar_type": "dsa",
      "title": "Solve: Two Sum",
      "estimated_minutes": 30,
      "priority": "high",
      "difficulty": "easy",
      "status": "pending",
      "sort_order": 1
    },
    {
      "id": "t-003",
      "plan_day_id": "pd-2026-07-01",
      "kind": "read",
      "item_type": "design_problem",
      "item_id": "design-url-shortener",
      "pillar_type": "system_design",
      "title": "Read: URL Shortener — requirements & capacity",
      "estimated_minutes": 60,
      "priority": "medium",
      "difficulty": "medium",
      "status": "pending",
      "sort_order": 2
    }
  ]
}
```

### 3.4 Submit revision recall

**Request**
```http
POST /api/v1/revision/rev-abc123/recall
Authorization: Bearer eyJhbGci...
Content-Type: application/json

{ "recall": "correct", "time_spent_minutes": 8, "notes": "Recalled hashing trick" }
```

**Response `200`** — interval advanced 1 → 3 days, next due computed:
```json
{
  "id": "rev-abc123",
  "user_id": "8c4a1e2b-7d4a-4b1e-9f2a-2b6c1d0e3f4a",
  "item_type": "topic",
  "item_id": "topic-arrays-hashing",
  "pillar_type": "dsa",
  "title": "Arrays & Hashing",
  "interval_days": 3,
  "stage": 1,
  "ease": 2.6,
  "due_at": "2026-07-05",
  "last_reviewed_at": "2026-07-02T09:15:00Z",
  "last_recall": "correct",
  "review_count": 2,
  "lapse_count": 0,
  "is_active": true
}
```
A `"recall": "incorrect"` resets `interval_days` to 1, `stage` to 0, and increments
`lapse_count` (per PRD §7.13: incorrect recall resets to +1; correct advances
through 1/3/7/15/30). Recall is binary — `correct | incorrect` (per D1).

---

## 4. Consistency guarantees

- Every response schema in `openapi.yaml` maps 1:1 to a table in
  `04-DATABASE-SCHEMA.md`; enum members are identical strings across the DB ENUM
  types, the OpenAPI `schemas`, and the tables above.
- Polymorphic `plan_tasks`/`revision_items` expose `(item_type, item_id)` exactly
  as stored; `item_type` is the shared `PlanItemType` enum.
- `confidence` is consistently 1–5; time fields are minutes everywhere.
