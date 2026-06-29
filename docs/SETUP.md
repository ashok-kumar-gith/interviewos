# InterviewOS — Local Development Setup

This guide gets the full stack running locally. There are **two supported paths**:

1. **Docker path** (`make dev`) — the recommended default; mirrors the deployed
   topology (Postgres, Redis, backend, frontend, Nginx, MailHog).
2. **Sudo-free `~/.local` toolchain** (`scripts/dev-local.sh`) — for machines
   without Docker/Homebrew/sudo, running Go + Node + Postgres + Redis from the
   user's home directory.

Both paths run the same code against the same migrations and seed.

---

## Prerequisites

| Tool | Docker path | Sudo-free path |
|------|-------------|----------------|
| Docker + Docker Compose v2 | required | not used |
| Go 1.24+ | optional (runs in container) | required (in `~/.local`) |
| Node 20+ | optional (runs in container) | required (in `~/.local`) |
| PostgreSQL 16 | container | compiled into `~/.local/pgsql` |
| Redis 7 | container | compiled into `~/.local/redis` |
| `make` | recommended | recommended |

---

## Path 1 — Docker (`make dev`)

From the repo root:

```bash
make dev        # build + boot: postgres, redis, mailhog, backend, frontend, nginx
```

This wraps `docker compose -f infra/docker-compose.yml up --build`. Services:

| Service  | Port(s)      | Notes |
|----------|--------------|-------|
| nginx    | 80           | edge proxy: `/` → frontend, `/api`,`/healthz`,`/readyz`,`/swagger` → backend |
| frontend | 3000         | Next.js |
| backend  | 8080         | Go API |
| postgres | 5432         | volume `pgdata` |
| redis    | 6379         | volume `redisdata` |
| mailhog  | 1025 (SMTP), 8025 (UI) | dev-only mail catcher |

Other useful targets:

```bash
make up      # start in the background
make logs    # tail logs
make ps      # status
make down    # stop + remove
```

Once up, apply migrations and seed (these run on the host against the container
DB, or `docker compose exec backend ...` if you prefer in-container):

```bash
make migrate     # apply all pending migrations
make seed        # load DSA + System Design curriculum (idempotent)
```

Open:

- App: http://localhost (via Nginx) or http://localhost:3000 (direct)
- API health: http://localhost:8080/healthz and `/readyz`
- Swagger: http://localhost:8080/swagger
- Metrics: http://localhost:8080/metrics
- MailHog UI: http://localhost:8025

---

## Path 2 — Sudo-free `~/.local` toolchain (`scripts/dev-local.sh`)

For environments without Docker. Go, Node, Postgres, and Redis live under
`~/.local`, with data dirs and ports isolated under the user's home. The helper
script sources `~/.local/interviewos-env.sh` (created during toolchain setup) and
sets the env vars below automatically.

```bash
scripts/dev-local.sh start-db    # start postgres (port 5433) + redis (6379)
scripts/dev-local.sh migrate     # apply migrations
scripts/dev-local.sh seed        # load curriculum seed
scripts/dev-local.sh backend     # run the Go API (foreground, :8080)
scripts/dev-local.sh frontend    # run the Next.js dev server (foreground, :3000)
scripts/dev-local.sh status      # show postgres / redis / backend status
scripts/dev-local.sh stop-db     # stop postgres + redis
```

Run `backend` and `frontend` in separate terminals. The script exports a working
`DATABASE_URL` (Postgres on **5433**, socket dir `/tmp`), `REDIS_URL`, `ENV`,
`PORT`, `LOG_LEVEL`, `JWT_SECRET`, and `CORS_ORIGINS` so the binaries find their
backing services. No Nginx in this path — hit the frontend (`:3000`) and backend
(`:8080`) directly.

> Note: this path uses Postgres port **5433** (vs. 5432 in Docker) to avoid
> clashing with any system Postgres.

---

## Environment variables

The backend is 12-factor: all config comes from the environment and is validated
at startup (it fails fast on missing/invalid values).

### Backend

| Var | Required | Example / default | Purpose |
|-----|----------|-------------------|---------|
| `PORT` | yes | `8080` | HTTP listen port |
| `DATABASE_URL` | yes | `postgres://interviewos:interviewos@localhost:5432/interviewos?sslmode=disable` | Postgres DSN |
| `REDIS_URL` | yes | `redis://localhost:6379/0` | Redis DSN |
| `JWT_SECRET` | yes | (long random) | JWT signing secret |
| `ENV` | yes | `development` / `production` | environment name |
| `LOG_LEVEL` | no | `debug` / `info` | log verbosity |
| `CORS_ORIGINS` | yes | `http://localhost:3000` | comma-separated allowlist |
| `ACCESS_TOKEN_TTL` | no | `15m` | access JWT lifetime |
| `REFRESH_TOKEN_TTL` | no | `720h` | refresh token lifetime |
| `RESET_TOKEN_TTL` | no | `1h` | password-reset token lifetime |
| `ANTHROPIC_API_KEY` | no | (key) | Claude API key; omit to run AI disabled |
| `AI_MODEL` | no | `claude-opus-4-8` | model id for AI features |
| `AI_ENABLED` | no | `true` / `false` | master AI feature flag |

### Frontend

| Var | Required | Example | Purpose |
|-----|----------|---------|---------|
| `NEXT_PUBLIC_API_BASE_URL` | yes | `http://localhost:8080` | backend base URL the SPA calls |

The Docker path sets these in `infra/docker-compose.yml`; the sudo-free path sets
them in `scripts/dev-local.sh`.

---

## Migrations & seed

Migrations live in `backend/migrations/` as `NNNNNN_name.up.sql` /
`.down.sql` pairs (golang-migrate). The runner is `backend/cmd/migrate`:

```bash
cd backend
go run ./cmd/migrate up          # apply all pending (default)
go run ./cmd/migrate down        # roll back one step
go run ./cmd/migrate down-all    # roll back everything
go run ./cmd/migrate version     # print current version
```

The seed loader (`backend/cmd/seed`) is idempotent — every entity is upserted by
its natural key, so it is safe to re-run:

```bash
cd backend && go run ./cmd/seed
```

`/readyz` returns ready only once migrations are applied and Postgres + Redis are
reachable.

---

## Running tests

**Backend unit tests** (fast, no external services):

```bash
make be-test                 # cd backend && go test ./...
```

**Backend integration tests** need a real Postgres + Redis and must run
**serially** (`-p 1`) because they share one database — parallel packages would
race on the same tables:

```bash
# bring up Postgres + Redis first (Docker path: `make up`; sudo-free: start-db),
# set DATABASE_URL / REDIS_URL, then:
make be-test-integration     # cd backend && go test ./... -p 1
```

**Frontend:**

```bash
make fe-typecheck            # tsc
make fe-lint                 # eslint
make fe-build                # next build (CI does `rm -rf .next` first)
```

See `docs/DEVELOPER.md` for the full testing strategy and the `-p 1` rationale.
