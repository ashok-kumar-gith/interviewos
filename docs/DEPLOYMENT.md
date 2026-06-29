# InterviewOS — Deployment Guide

Three deployment options, from simplest to most scalable:

1. **Docker Compose** — single host, prod-parity.
2. **Kubernetes (raw manifests + Kustomize)** — `infra/k8s/`.
3. **Helm chart** — `infra/helm/interviewos/`, parameterized per environment.

All three deploy the same two images:

- `interviewos-backend` — Go API (also ships the `migrate` and `seed` runners)
- `interviewos-frontend` — Next.js (standalone output)

Build them from the repo root:

```bash
docker build -t interviewos-backend:latest ./backend
docker build -t interviewos-frontend:latest ./frontend
```

> Image note: the K8s/Helm migrate & seed Jobs invoke `/app/migrate` and
> `/app/seed`. Ensure the backend image includes those binaries (build all three
> `cmd/` entrypoints), or override the Job command to your own runner. The API
> container itself only needs `/app/api`.

---

## Health, readiness, and observability

| Endpoint | Meaning |
|----------|---------|
| `GET /healthz` | Liveness — process is up. Used by the liveness probe. |
| `GET /readyz` | Readiness — Postgres + Redis reachable **and** migrations applied. Gates load-balancer membership and rollout progression. |
| `GET /metrics` | Prometheus metrics (RED per route, DB pool, cache hit ratio, AI tokens/cost). |
| `GET /swagger` | Live Swagger UI for the API. |

Rollouts wait on `/readyz`; graceful shutdown drains in-flight requests on
SIGTERM so rolling deploys lose no requests.

---

## Option 1 — Docker Compose (production on one host)

`infra/docker-compose.yml` defines the full topology (postgres, redis, backend,
frontend, nginx, mailhog). For production:

1. Override the dev defaults — set a strong `JWT_SECRET`, real `DATABASE_URL`
   credentials, `CORS_ORIGINS`, and `NEXT_PUBLIC_API_BASE_URL` (via an env file
   or your orchestrator's secret mechanism). **Do not ship `JWT_SECRET:
   dev-only-change-me`.**
2. Terminate TLS at Nginx (mount certs into the nginx service) or in front of it.
3. Bring it up and run migrations + seed:

```bash
docker compose -f infra/docker-compose.yml up -d --build
docker compose -f infra/docker-compose.yml exec backend /app/migrate up
docker compose -f infra/docker-compose.yml exec backend /app/seed
```

Nginx (`infra/nginx/nginx.conf`) routes `/` to the frontend and `/api`,
`/healthz`, `/readyz`, `/swagger` to the backend.

> MailHog is a dev-only mail catcher; drop it (or replace SMTP config) in
> production. At GA notifications are in-app only.

---

## Option 2 — Kubernetes with kubectl + Kustomize

Manifests in `infra/k8s/` (namespace `interviewos`):

| File | Resource |
|------|----------|
| `namespace.yaml` | Namespace |
| `configmap.yaml` | non-secret env |
| `secret.yaml` | JWT secret, DB creds, Anthropic key (**placeholders**) |
| `postgres.yaml` | Postgres StatefulSet + headless Service + 10Gi PVC |
| `redis.yaml` | Redis Deployment + Service |
| `migrate-job.yaml` | migrate Job + optional seed Job |
| `backend.yaml` | backend Deployment (probes, resources) + Service |
| `frontend.yaml` | frontend Deployment + Service |
| `ingress.yaml` | nginx Ingress (`/` → frontend, `/api…` → backend) |
| `kustomization.yaml` | ties it together; pins image tags |

### Deploy

```bash
# 1. Make images available to the cluster.
#    - Remote registry: push them and set kustomization `images:` to the path+tag.
#    - Local kind:      kind load docker-image interviewos-backend:latest interviewos-frontend:latest
#    - Local minikube:  eval $(minikube docker-env) && docker build ...

# 2. Override the placeholder Secret BEFORE applying (sealed-secrets,
#    external-secrets, or a one-off):
kubectl create namespace interviewos
kubectl -n interviewos create secret generic interviewos-secrets \
  --from-literal=JWT_SECRET="$(openssl rand -hex 32)" \
  --from-literal=POSTGRES_USER=interviewos \
  --from-literal=POSTGRES_PASSWORD="$(openssl rand -hex 24)" \
  --from-literal=POSTGRES_DB=interviewos \
  --from-literal=DATABASE_URL="postgres://interviewos:<pw>@interviewos-postgres:5432/interviewos?sslmode=disable" \
  --from-literal=ANTHROPIC_API_KEY="" \
  --dry-run=client -o yaml | kubectl apply -f -

# 3. Apply everything (skip secret.yaml if you created the Secret above, or just
#    let kustomize own it and edit secret.yaml first).
kubectl apply -k infra/k8s

# Dry-run only (no cluster mutation):
kubectl apply --dry-run=client -k infra/k8s
```

The migrate Job runs `cmd/migrate up`; the seed Job is idempotent. Update the
`host` in `ingress.yaml` and `CORS_ORIGINS`/`NEXT_PUBLIC_API_BASE_URL` in
`configmap.yaml` to your domain. For TLS, add cert-manager + a `tls:` block to
the Ingress.

Convenience targets: `make k8s-apply`, `make k8s-dryrun`, `make k8s-delete`.

---

## Option 3 — Helm

Chart at `infra/helm/interviewos/`. Parameterizes replicas, resources, image
tags, ingress host, secrets, and the in-cluster Postgres/Redis toggles.

### Render / lint (no cluster needed)

```bash
helm template interviewos infra/helm/interviewos
make helm-template
```

### Install / upgrade

```bash
helm upgrade --install interviewos infra/helm/interviewos \
  --namespace interviewos --create-namespace \
  --set secrets.jwtSecret="$(openssl rand -hex 32)" \
  --set secrets.postgresPassword="$(openssl rand -hex 24)" \
  --set ingress.host=interviewos.yourdomain.com \
  --set config.corsOrigins=https://interviewos.yourdomain.com \
  --set config.apiBaseUrl=https://interviewos.yourdomain.com \
  --set backend.image.repository=registry.example.com/interviewos-backend \
  --set backend.image.tag="$GIT_SHA" \
  --set frontend.image.repository=registry.example.com/interviewos-frontend \
  --set frontend.image.tag="$GIT_SHA"

make helm-install     # uses chart defaults
```

The migrate Job runs as a `pre-install,pre-upgrade` Helm hook (weight 0), and the
seed Job runs after it (weight 1) before the app workloads roll.

### Key `values.yaml` settings

| Value | Default | Purpose |
|-------|---------|---------|
| `backend.replicaCount` / `frontend.replicaCount` | `2` | replicas |
| `backend.resources` / `frontend.resources` | requests+limits | CPU/memory |
| `backend.readinessPath` / `livenessPath` | `/readyz` / `/healthz` | probes |
| `postgres.enabled` | `true` | set `false` to use a managed DB (then set `secrets.databaseUrl`) |
| `redis.enabled` | `true` | set `false` to use a managed Redis (then set `config.redisUrl`) |
| `postgres.storage` | `10Gi` | PVC size |
| `ingress.enabled` / `ingress.host` | `true` / `interviewos.example.com` | routing |
| `ingress.tls.enabled` / `tls.secretName` | `false` | TLS via cert-manager |
| `migrate.enabled` / `seed.enabled` | `true` | lifecycle Jobs |
| `secrets.create` | `true` | set `false` to supply your own Secret `<release>-secrets` |
| `config.*` | env values | ENV, LOG_LEVEL, TTLs, AI_MODEL, AI_ENABLED, etc. |

### Uninstall

```bash
helm uninstall interviewos --namespace interviewos
make helm-uninstall
```

---

## Environment & secrets

- **Never commit real secrets.** `infra/k8s/secret.yaml` and the chart's
  `secrets.*` defaults are placeholders.
- Production secrets: GitHub Actions secrets in CI, and a sealed-secrets /
  external-secrets operator (or `kubectl create secret`) in-cluster.
- Required secrets: `JWT_SECRET`, `DATABASE_URL` (or split Postgres creds when
  using the in-cluster DB), and optionally `ANTHROPIC_API_KEY` (omit to run with
  AI disabled — the deterministic engines are the source of truth either way).
- Non-secret config (TTLs, CORS origins, AI model, API base URL) lives in the
  ConfigMap / `config.*` values.

---

## CI/CD

`.github/workflows/ci.yml` runs on push/PR to `main`:

- **backend** — `go vet`, `go build`, unit `go test -race` + coverage artifact.
- **backend-integration** — Postgres 16 + Redis 7 service containers, applies
  migrations, then `go test ./... -p 1` (serial) + coverage artifact.
- **frontend** — typecheck, lint, `rm -rf .next`, build.
- **openapi** — `openapi-spec-validator` against `backend/api/openapi.yaml`.
- **docker** — builds both images (gated on the test jobs).

A CD pipeline should build + push images tagged by SHA, run the migrate Job, then
`kubectl apply -k` / `helm upgrade`, followed by a smoke test against `/readyz`.
