-- 000005_resume.up.sql
-- Resume module schema: resume_profiles (one per user) and resume_projects
-- per docs/04-DATABASE-SCHEMA.md §4.3. User-data tables: soft delete + updated_at
-- trigger. Depends on 000001_auth (users table + set_updated_at()).

BEGIN;

-- resume_profiles -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS resume_profiles (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    headline         TEXT NULL,
    summary          TEXT NULL,
    years_experience NUMERIC(4,1) NULL,
    skills           JSONB NOT NULL DEFAULT '[]',
    target_keywords  JSONB NOT NULL DEFAULT '[]',
    ats_score        NUMERIC(5,2) NULL,
    last_scored_at   TIMESTAMPTZ NULL,
    ai_feedback      JSONB NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ NULL
);

-- One active resume profile per user; soft-deleted rows free the slot.
CREATE UNIQUE INDEX IF NOT EXISTS uq_resume_profile_user_active
    ON resume_profiles (user_id) WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_resume_profiles_updated_at ON resume_profiles;
CREATE TRIGGER trg_resume_profiles_updated_at
    BEFORE UPDATE ON resume_profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- resume_projects -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS resume_projects (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_profile_id UUID NOT NULL REFERENCES resume_profiles (id) ON DELETE CASCADE,
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    role              TEXT NULL,
    description       TEXT NULL,
    impact            TEXT NULL,
    metrics           JSONB NOT NULL DEFAULT '[]',
    tech_stack        JSONB NOT NULL DEFAULT '[]',
    start_date        DATE NULL,
    end_date          DATE NULL,
    sort_order        INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_rproj_profile ON resume_projects (resume_profile_id);
CREATE INDEX IF NOT EXISTS idx_rproj_user ON resume_projects (user_id);

DROP TRIGGER IF EXISTS trg_resume_projects_updated_at ON resume_projects;
CREATE TRIGGER trg_resume_projects_updated_at
    BEFORE UPDATE ON resume_projects
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
