# Deploy InterviewOS for free (Neon + Render + Vercel)

A $0, publicly-accessible deployment:

| Layer | Service | Free tier |
|-------|---------|-----------|
| Postgres | **Neon** | serverless, permanent free tier |
| Backend (Go/Gin) | **Render** (Docker web service) | free — sleeps after ~15 min idle (~30–50 s cold start) |
| Frontend (Next.js) | **Vercel** (Hobby) | free, no sleep |
| Redis | *omitted* | backend degrades gracefully; add Upstash later if you want |

**Architecture:** the Vercel frontend **proxies `/api/*` to the Render backend**
(`frontend/vercel.json` rewrites), so the browser sees everything on one origin —
no CORS setup and the auth cookie (SameSite=Strict) just works. Nothing else to wire.

> ⚠️ The code runner (`/code/run`) stays disabled in production (`CODE_RUNNER_ENABLED`
> unset) — it executes arbitrary code with server privileges. Do not enable it publicly.

---

## 1. Database — Neon (2 min)

1. Sign up at **https://neon.tech** → **Create project** (name `interviewos`, region near you).
2. Copy the **connection string** (looks like `postgresql://user:pass@ep-xxx.aws.neon.tech/neondb?sslmode=require`). Keep the `?sslmode=require`.
3. From your machine, initialize the schema + seed against Neon:
   ```bash
   cd ~/interviewos/backend
   DATABASE_URL="<neon-url>" go run ./cmd/migrate up
   DATABASE_URL="<neon-url>" go run ./cmd/seed
   # optional: load your imported questions
   cd ~/interviewos
   python3 scripts/import_questions.py "/path/Questions list.xlsx" | psql "<neon-url>"
   ```
   (Render's free tier has no shell/jobs, so this one-time setup runs locally against the same DB.)

## 2. Backend — Render (5 min)

1. Sign up at **https://render.com** and connect your GitHub (`ashok-kumar-gith/interviewos`).
2. **New → Blueprint** → pick the repo. Render reads `render.yaml` and proposes the
   `interviewos-api` Docker web service (free plan).
3. Before/after first deploy, set the env vars marked `sync: false`:
   - **`DATABASE_URL`** = your Neon URL from step 1.
   - **`CORS_ORIGINS`** = your Vercel URL (fill in after step 3, e.g. `https://interviewos.vercel.app`).
   - `JWT_SECRET` is auto-generated; `ENV=production`, TTLs, `AI_ENABLED=false` come from the blueprint.
4. Deploy. When it's live, note the URL: **`https://interviewos-api-XXXX.onrender.com`**.
   Verify: open `…/healthz` → `{"status":"ok"}` and `…/swagger`.

## 3. Frontend — Vercel (5 min)

1. Sign up at **https://vercel.com** → **Add New → Project** → import the repo.
2. **Root Directory:** set to **`frontend`** (important — the Next.js app lives there).
3. Framework preset: **Next.js** (auto-detected). Leave build/output defaults.
4. Wire the proxy to your backend: edit **`frontend/vercel.json`** and replace every
   `REPLACE_WITH_RENDER_HOST` with your Render host (e.g. `interviewos-api-XXXX.onrender.com`),
   then commit + push:
   ```bash
   cd ~/interviewos
   sed -i '' 's/REPLACE_WITH_RENDER_HOST/interviewos-api-XXXX.onrender.com/g' frontend/vercel.json
   git add frontend/vercel.json && git commit -m "chore: point vercel proxy at render backend" && git push
   ```
   (Vercel redeploys on push.)
5. **Env var (optional):** leave `NEXT_PUBLIC_API_BASE_URL` **unset** — the client then
   uses the current origin, and the vercel.json rewrite proxies `/api/*` to Render.
6. Once deployed, go back to Render and set **`CORS_ORIGINS`** to the Vercel URL.

## 4. Verify

- Open your Vercel URL → you land on `/login`.
- **Register** an account (email/password — OAuth stays a 501 stub until you add Google/GitHub creds).
- You should reach the dashboard; complete intake → roadmap → today all work.
- Browse `/problems` (your seeded + imported questions), `/revision`, etc.

## Notes & gotchas

- **Cold starts:** the Render free backend sleeps after ~15 min idle; the first request
  after that takes ~30–50 s (the frontend shows loading states). Upgrade the Render plan
  or switch to Cloud Run to avoid this.
- **Make yourself admin** (to use `/admin/content`): `psql "<neon-url>" -c "UPDATE users SET role='admin' WHERE email='you@example.com';"` then log out/in.
- **AI features** return deterministic fallbacks until you set `ANTHROPIC_API_KEY` (and `AI_ENABLED=true`) on Render.
- **Custom domain / no cold start:** move the backend to Google Cloud Run (scale-to-zero,
  fast cold start, free monthly allowance) — same Docker image; point vercel.json at it.
