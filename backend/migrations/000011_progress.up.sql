-- 000011_progress.up.sql
-- Progress tracking & activity persistence per docs/04-DATABASE-SCHEMA.md §4.3:
--   user_topic_progress, user_problem_progress, study_sessions, streak_days.
-- These tables are mutated transactionally on task completion (internal/progress)
-- and feed the Today view and the Dashboard aggregate (readiness / streak / time).
--
-- Self-contained references (mirroring 000006_roadmap): topic_id / problem_id /
-- plan_task_id reference content/roadmap tables; user_id keeps its hard FK to
-- users(id) so the user-owned cascade holds. Cross-module references that would
-- otherwise be ON DELETE RESTRICT/CASCADE are stored as plain UUIDs WITHOUT hard
-- FKs to keep this migration independently applicable; integrity is validated at
-- the service layer (consistent with the polymorphic note in §4.3).

BEGIN;

-- progress_status enum (introduced here; absent in 000001-000010).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'progress_status') THEN
        CREATE TYPE progress_status AS ENUM ('not_started','in_progress','completed','needs_review');
    END IF;
END
$$;

-- user_topic_progress: per-user mastery of a content topic.
CREATE TABLE IF NOT EXISTS user_topic_progress (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    topic_id            UUID NOT NULL,
    status              progress_status NOT NULL DEFAULT 'not_started',
    confidence          SMALLINT NULL CHECK (confidence BETWEEN 1 AND 5),
    time_spent_minutes  INTEGER NOT NULL DEFAULT 0,
    times_revised       INTEGER NOT NULL DEFAULT 0,
    last_studied_at     TIMESTAMPTZ NULL,
    first_completed_at  TIMESTAMPTZ NULL,
    notes               TEXT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_utp_user_topic
    ON user_topic_progress (user_id, topic_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_utp_user_status ON user_topic_progress (user_id, status);
CREATE INDEX IF NOT EXISTS idx_utp_confidence ON user_topic_progress (user_id, confidence);

-- user_problem_progress: per-user solve state of a DSA problem.
CREATE TABLE IF NOT EXISTS user_problem_progress (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    problem_id          UUID NOT NULL,
    status              progress_status NOT NULL DEFAULT 'not_started',
    confidence          SMALLINT NULL CHECK (confidence BETWEEN 1 AND 5),
    attempts            INTEGER NOT NULL DEFAULT 0,
    solved              BOOLEAN NOT NULL DEFAULT false,
    time_spent_minutes  INTEGER NOT NULL DEFAULT 0,
    last_attempt_at     TIMESTAMPTZ NULL,
    solved_at           TIMESTAMPTZ NULL,
    notes               TEXT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_upp_user_problem
    ON user_problem_progress (user_id, problem_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_upp_user_status ON user_problem_progress (user_id, status);
CREATE INDEX IF NOT EXISTS idx_upp_solved ON user_problem_progress (user_id, solved);

-- study_sessions: time tracking (feeds time-spent heatmap & analytics).
CREATE TABLE IF NOT EXISTS study_sessions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    plan_task_id      UUID NULL,
    pillar_type       pillar_type NULL,
    started_at        TIMESTAMPTZ NOT NULL,
    ended_at          TIMESTAMPTZ NULL,
    duration_minutes  INTEGER NOT NULL DEFAULT 0,
    source            TEXT NOT NULL DEFAULT 'timer' CHECK (source IN ('timer','manual')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_session_user_start ON study_sessions (user_id, started_at);
CREATE INDEX IF NOT EXISTS idx_session_task ON study_sessions (plan_task_id);

-- streak_days: one row per active study day per user (drives the streak).
CREATE TABLE IF NOT EXISTS streak_days (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    date             DATE NOT NULL,
    tasks_completed  INTEGER NOT NULL DEFAULT 0,
    minutes_studied  INTEGER NOT NULL DEFAULT 0,
    goal_met         BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_streak_user_date
    ON streak_days (user_id, date) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_streak_user_date ON streak_days (user_id, date);

-- Maintain updated_at (set_updated_at() created by migration 000001_auth).
DROP TRIGGER IF EXISTS trg_utp_updated_at ON user_topic_progress;
CREATE TRIGGER trg_utp_updated_at
    BEFORE UPDATE ON user_topic_progress
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_upp_updated_at ON user_problem_progress;
CREATE TRIGGER trg_upp_updated_at
    BEFORE UPDATE ON user_problem_progress
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_study_sessions_updated_at ON study_sessions;
CREATE TRIGGER trg_study_sessions_updated_at
    BEFORE UPDATE ON study_sessions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_streak_days_updated_at ON streak_days;
CREATE TRIGGER trg_streak_days_updated_at
    BEFORE UPDATE ON streak_days
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
