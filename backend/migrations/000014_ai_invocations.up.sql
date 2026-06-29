-- 000014_ai_invocations.up.sql
-- AI assistants (M4) persistence per docs/04-DATABASE-SCHEMA.md (ai_invocations)
-- and docs/03-ARCHITECTURE.md §9 (AI orchestration: call Claude, cache, graceful
-- fallback to deterministic, cost controls). Every /ai/* call records exactly one
-- row here for cost tracking, observability, and the graceful-fallback audit
-- trail (used_fallback flag).
--
-- Self-contained references (mirroring 000011/000013): user_id keeps a hard FK to
-- users(id) so the user-owned cascade holds. The ai_feature and
-- ai_invocation_status enums are created here if not already present.

BEGIN;

-- ai_feature: the AI assistant feature that produced this invocation.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'ai_feature') THEN
        CREATE TYPE ai_feature AS ENUM (
            'planner', 'coach', 'resume_review', 'story_improve',
            'weakness_detect', 'daily_plan', 'sd_review'
        );
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'ai_invocation_status') THEN
        CREATE TYPE ai_invocation_status AS ENUM (
            'pending', 'succeeded', 'failed', 'fallback'
        );
    END IF;
END$$;

-- ai_invocations: one row per AI feature call (real Claude call or deterministic
-- fallback). prompt/completion token counts are nullable (unknown for fallback).
CREATE TABLE IF NOT EXISTS ai_invocations (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    feature           ai_feature NOT NULL,
    status            ai_invocation_status NOT NULL DEFAULT 'pending',
    model             TEXT NULL,
    prompt_tokens     INTEGER NULL,
    completion_tokens INTEGER NULL,
    used_fallback     BOOLEAN NOT NULL DEFAULT false,
    latency_ms        INTEGER NULL,
    error             TEXT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Hot path: a user's recent invocations (cost dashboards, per-user budgets).
CREATE INDEX IF NOT EXISTS idx_ai_user_created
    ON ai_invocations (user_id, created_at DESC);
-- Per-feature rollups (spend by feature).
CREATE INDEX IF NOT EXISTS idx_ai_user_feature
    ON ai_invocations (user_id, feature);

COMMIT;
