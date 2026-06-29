-- 000004_behavioral.up.sql
-- Behavioral module schema: the story_theme enum and the behavioral_stories
-- table (STAR story builder) per docs/04-DATABASE-SCHEMA.md §3/§4.3.
--
-- behavioral_stories is a user-data table: it carries deleted_at (soft delete)
-- and FK→users(id) ON DELETE CASCADE. The shared set_updated_at() trigger
-- function is created by migration 000001; we (re)attach it here defensively.

BEGIN;

-- Enums ---------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'story_theme') THEN
        CREATE TYPE story_theme AS ENUM (
            'leadership',
            'ownership',
            'conflict',
            'failure',
            'mentorship',
            'stakeholder_management',
            'project_rescue',
            'production_incident',
            'ambiguity',
            'impact'
        );
    END IF;
END$$;

-- Defensive: ensure the shared updated_at trigger function exists even if this
-- migration is applied in isolation against a fresh database.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- behavioral_stories --------------------------------------------------------
CREATE TABLE IF NOT EXISTS behavioral_stories (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title          TEXT NOT NULL,
    theme          story_theme NOT NULL,
    situation      TEXT NULL,
    task           TEXT NULL,
    action         TEXT NULL,
    result         TEXT NULL,
    metrics        TEXT NULL,
    tags           JSONB NOT NULL DEFAULT '[]',
    ai_improved    BOOLEAN NOT NULL DEFAULT false,
    ai_feedback    JSONB NULL,
    strength_score NUMERIC(5,2) NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_story_user_theme
    ON behavioral_stories (user_id, theme);
-- Soft-delete-aware lookups by owner.
CREATE INDEX IF NOT EXISTS idx_story_user
    ON behavioral_stories (user_id) WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_behavioral_stories_updated_at ON behavioral_stories;
CREATE TRIGGER trg_behavioral_stories_updated_at
    BEFORE UPDATE ON behavioral_stories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
