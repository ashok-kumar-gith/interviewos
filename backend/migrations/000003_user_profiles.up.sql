-- 000003_user_profiles.up.sql
-- Intake / profile schema: the user_profiles table per docs/04-DATABASE-SCHEMA.md
-- §4.1. One active profile per user (intake answers driving the Curriculum
-- Engine).
--
-- Self-contained references: the schema doc models target_company_id as an FK to
-- companies(id) and track_id as an FK to tracks(id). Neither the companies nor
-- the tracks table exists in this worktree (they are introduced by other
-- migrations). To keep this migration self-contained and independently
-- applicable, both columns are stored as plain UUIDs WITHOUT a hard FK. They are
-- soft references: integrity is validated at the application layer, and a later
-- migration may attach the FKs once those parent tables exist.

BEGIN;

CREATE TABLE IF NOT EXISTS user_profiles (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    track_id                UUID NOT NULL,
    years_experience        NUMERIC(4,1) NOT NULL DEFAULT 0 CHECK (years_experience >= 0),
    target_company_id       UUID NULL,
    target_role             TEXT NOT NULL,
    target_level            TEXT NULL,
    hours_per_week          SMALLINT NOT NULL DEFAULT 15 CHECK (hours_per_week BETWEEN 1 AND 80),
    start_date              DATE NOT NULL,
    target_weeks            SMALLINT NOT NULL DEFAULT 12 CHECK (target_weeks BETWEEN 1 AND 52),
    pillar_strengths        JSONB NOT NULL DEFAULT '{}',
    timezone                TEXT NOT NULL DEFAULT 'UTC',
    onboarding_completed_at TIMESTAMPTZ NULL,
    intake_answers          JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at              TIMESTAMPTZ NULL
);

-- One active (non-deleted) profile per user; a soft-deleted row frees the slot.
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_profiles_user_active
    ON user_profiles (user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_profile_company
    ON user_profiles (target_company_id);

-- Maintain updated_at (set_updated_at() is created by migration 000001_auth).
DROP TRIGGER IF EXISTS trg_user_profiles_updated_at ON user_profiles;
CREATE TRIGGER trg_user_profiles_updated_at
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
