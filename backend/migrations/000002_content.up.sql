-- 000002_content.up.sql
-- Content / Curriculum data layer per docs/04-DATABASE-SCHEMA.md §4.2.
-- Tables: tracks, pillars, topics, subtopics, resources, topic_resources,
-- patterns, problems, problem_patterns, problem_sources, companies,
-- company_weights, problem_company_frequency.
--
-- Content tables are seeded/migration-managed. They carry deleted_at where the
-- schema permits retiring content without breaking FK history; join tables that
-- the schema does not mark soft-deletable omit it.

BEGIN;

-- Enums (content) -----------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'pillar_type') THEN
        CREATE TYPE pillar_type AS ENUM ('dsa', 'system_design', 'lld', 'backend_engineering', 'behavioral', 'resume');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'resource_type') THEN
        CREATE TYPE resource_type AS ENUM ('book', 'video', 'article', 'course', 'github', 'practice', 'documentation', 'blog', 'cheatsheet');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'difficulty') THEN
        CREATE TYPE difficulty AS ENUM ('easy', 'medium', 'hard');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'priority') THEN
        CREATE TYPE priority AS ENUM ('low', 'medium', 'high', 'critical');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'problem_source_name') THEN
        CREATE TYPE problem_source_name AS ENUM ('blind75', 'neetcode150', 'grind75', 'tech_interview_handbook', 'leetcode_top', 'striver_sde', 'custom');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'problem_platform') THEN
        CREATE TYPE problem_platform AS ENUM ('leetcode', 'hackerrank', 'codeforces', 'interviewbit', 'gfg', 'custom');
    END IF;
END$$;

-- tracks --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tracks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NULL,
    seniority   TEXT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_tracks_slug ON tracks (slug);

DROP TRIGGER IF EXISTS trg_tracks_updated_at ON tracks;
CREATE TRIGGER trg_tracks_updated_at BEFORE UPDATE ON tracks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- pillars -------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pillars (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    track_id    UUID NOT NULL REFERENCES tracks (id) ON DELETE RESTRICT,
    type        pillar_type NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NULL,
    weight      NUMERIC(5,2) NOT NULL DEFAULT 1.0,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pillars_track_type ON pillars (track_id, type);
CREATE INDEX IF NOT EXISTS idx_pillar_track ON pillars (track_id);

DROP TRIGGER IF EXISTS trg_pillars_updated_at ON pillars;
CREATE TRIGGER trg_pillars_updated_at BEFORE UPDATE ON pillars
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- topics --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS topics (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pillar_id          UUID NOT NULL REFERENCES pillars (id) ON DELETE RESTRICT,
    track_id           UUID NOT NULL REFERENCES tracks (id) ON DELETE RESTRICT,
    slug               TEXT NOT NULL,
    name               TEXT NOT NULL,
    summary            TEXT NULL,
    concept_md         TEXT NULL,
    difficulty         difficulty NOT NULL DEFAULT 'medium',
    priority           priority NOT NULL DEFAULT 'medium',
    estimated_hours    NUMERIC(5,2) NOT NULL DEFAULT 2.0,
    common_mistakes    TEXT NULL,
    expected_questions JSONB NOT NULL DEFAULT '[]',
    prerequisites      JSONB NOT NULL DEFAULT '[]',
    sort_order         INTEGER NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_topics_track_slug ON topics (track_id, slug);
CREATE INDEX IF NOT EXISTS idx_topic_pillar ON topics (pillar_id);
CREATE INDEX IF NOT EXISTS idx_topic_track_diff ON topics (track_id, difficulty);
CREATE INDEX IF NOT EXISTS idx_topic_questions ON topics USING GIN (expected_questions);

DROP TRIGGER IF EXISTS trg_topics_updated_at ON topics;
CREATE TRIGGER trg_topics_updated_at BEFORE UPDATE ON topics
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- subtopics -----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS subtopics (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id        UUID NOT NULL REFERENCES topics (id) ON DELETE CASCADE,
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    content_md      TEXT NULL,
    estimated_hours NUMERIC(5,2) NOT NULL DEFAULT 0.5,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_subtopics_topic_slug ON subtopics (topic_id, slug);
CREATE INDEX IF NOT EXISTS idx_subtopic_topic ON subtopics (topic_id);

DROP TRIGGER IF EXISTS trg_subtopics_updated_at ON subtopics;
CREATE TRIGGER trg_subtopics_updated_at BEFORE UPDATE ON subtopics
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- resources -----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS resources (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type              resource_type NOT NULL,
    title             TEXT NOT NULL,
    author            TEXT NULL,
    url               TEXT NULL,
    provider          TEXT NULL,
    description       TEXT NULL,
    estimated_minutes INTEGER NULL,
    difficulty        difficulty NULL,
    priority          priority NOT NULL DEFAULT 'medium',
    is_free           BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_resources_url ON resources (url) WHERE url IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_resource_type ON resources (type);

DROP TRIGGER IF EXISTS trg_resources_updated_at ON resources;
CREATE TRIGGER trg_resources_updated_at BEFORE UPDATE ON resources
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- topic_resources (M:N) -----------------------------------------------------
CREATE TABLE IF NOT EXISTS topic_resources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id    UUID NOT NULL REFERENCES topics (id) ON DELETE CASCADE,
    resource_id UUID NOT NULL REFERENCES resources (id) ON DELETE RESTRICT,
    relevance   priority NOT NULL DEFAULT 'medium',
    is_primary  BOOLEAN NOT NULL DEFAULT false,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_topic_resources ON topic_resources (topic_id, resource_id);
CREATE INDEX IF NOT EXISTS idx_tr_resource ON topic_resources (resource_id);

DROP TRIGGER IF EXISTS trg_topic_resources_updated_at ON topic_resources;
CREATE TRIGGER trg_topic_resources_updated_at BEFORE UPDATE ON topic_resources
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- patterns ------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS patterns (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    track_id    UUID NOT NULL REFERENCES tracks (id) ON DELETE RESTRICT,
    slug        TEXT NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NULL,
    when_to_use TEXT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_patterns_slug ON patterns (slug);

DROP TRIGGER IF EXISTS trg_patterns_updated_at ON patterns;
CREATE TRIGGER trg_patterns_updated_at BEFORE UPDATE ON patterns
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- problems ------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS problems (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    track_id          UUID NOT NULL REFERENCES tracks (id) ON DELETE RESTRICT,
    topic_id          UUID NULL REFERENCES topics (id) ON DELETE SET NULL,
    slug              TEXT NOT NULL,
    title             TEXT NOT NULL,
    difficulty        difficulty NOT NULL,
    platform          problem_platform NOT NULL DEFAULT 'leetcode',
    external_id       TEXT NULL,
    url               TEXT NULL,
    prompt_summary    TEXT NULL,
    approach_md       TEXT NULL,
    common_mistakes   TEXT NULL,
    estimated_minutes INTEGER NOT NULL DEFAULT 30,
    frequency_score   NUMERIC(5,2) NOT NULL DEFAULT 0,
    is_premium        BOOLEAN NOT NULL DEFAULT false,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_problems_slug ON problems (slug);
CREATE INDEX IF NOT EXISTS idx_problem_topic ON problems (topic_id);
CREATE INDEX IF NOT EXISTS idx_problem_track_diff ON problems (track_id, difficulty);
CREATE INDEX IF NOT EXISTS idx_problem_freq ON problems (frequency_score DESC);

DROP TRIGGER IF EXISTS trg_problems_updated_at ON problems;
CREATE TRIGGER trg_problems_updated_at BEFORE UPDATE ON problems
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- problem_patterns (M:N) ----------------------------------------------------
CREATE TABLE IF NOT EXISTS problem_patterns (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    problem_id UUID NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    pattern_id UUID NOT NULL REFERENCES patterns (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_problem_patterns ON problem_patterns (problem_id, pattern_id);
CREATE INDEX IF NOT EXISTS idx_pp_pattern ON problem_patterns (pattern_id);

DROP TRIGGER IF EXISTS trg_problem_patterns_updated_at ON problem_patterns;
CREATE TRIGGER trg_problem_patterns_updated_at BEFORE UPDATE ON problem_patterns
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- problem_sources -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS problem_sources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    problem_id  UUID NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    source      problem_source_name NOT NULL,
    source_rank INTEGER NULL,
    source_url  TEXT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_problem_sources ON problem_sources (problem_id, source);
CREATE INDEX IF NOT EXISTS idx_ps_source ON problem_sources (source);

DROP TRIGGER IF EXISTS trg_problem_sources_updated_at ON problem_sources;
CREATE TRIGGER trg_problem_sources_updated_at BEFORE UPDATE ON problem_sources
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- companies -----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS companies (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug               TEXT NOT NULL,
    name               TEXT NOT NULL,
    logo_url           TEXT NULL,
    description        TEXT NULL,
    interview_style_md TEXT NULL,
    is_fully_weighted  BOOLEAN NOT NULL DEFAULT false,
    sort_order         INTEGER NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_companies_slug ON companies (slug);

DROP TRIGGER IF EXISTS trg_companies_updated_at ON companies;
CREATE TRIGGER trg_companies_updated_at BEFORE UPDATE ON companies
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- company_weights -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS company_weights (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id        UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    pillar_id         UUID NULL REFERENCES pillars (id) ON DELETE CASCADE,
    topic_id          UUID NULL REFERENCES topics (id) ON DELETE CASCADE,
    weight_multiplier NUMERIC(5,2) NOT NULL DEFAULT 1.0,
    note              TEXT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_cw_target CHECK (pillar_id IS NOT NULL OR topic_id IS NOT NULL)
);
-- A partial-index pair gives uniqueness whether topic_id is NULL (pillar weight)
-- or set, since NULLs are distinct in a plain UNIQUE index.
CREATE UNIQUE INDEX IF NOT EXISTS uq_cw_company_pillar
    ON company_weights (company_id, pillar_id) WHERE topic_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_cw_company_topic
    ON company_weights (company_id, topic_id) WHERE topic_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cw_company ON company_weights (company_id);
CREATE INDEX IF NOT EXISTS idx_cw_topic ON company_weights (topic_id);

DROP TRIGGER IF EXISTS trg_company_weights_updated_at ON company_weights;
CREATE TRIGGER trg_company_weights_updated_at BEFORE UPDATE ON company_weights
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- problem_company_frequency -------------------------------------------------
CREATE TABLE IF NOT EXISTS problem_company_frequency (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    problem_id       UUID NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    company_id       UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    frequency        NUMERIC(5,2) NOT NULL DEFAULT 0,
    last_seen_period TEXT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pcf_problem_company ON problem_company_frequency (problem_id, company_id);
CREATE INDEX IF NOT EXISTS idx_pcf_company_freq ON problem_company_frequency (company_id, frequency DESC);

DROP TRIGGER IF EXISTS trg_pcf_updated_at ON problem_company_frequency;
CREATE TRIGGER trg_pcf_updated_at BEFORE UPDATE ON problem_company_frequency
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
