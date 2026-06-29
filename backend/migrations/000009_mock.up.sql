-- 000009_mock.up.sql
-- Mock interview module schema: the mock_type, mock_outcome, and
-- finding_severity enums plus the mock_interviews and mock_findings tables per
-- docs/04-DATABASE-SCHEMA.md §"mock_interviews"/"mock_findings".
--
-- Both tables are user-data tables: they carry deleted_at (soft delete) and
-- FK→users(id) ON DELETE CASCADE. The shared set_updated_at() trigger function
-- is created by migration 000001; we (re)create it defensively here.
--
-- The full schema references topics, companies, design_problems, and plan_tasks
-- via optional FKs. topics and companies exist by this point (000002_content);
-- design_problem_id and remediation_task_id are kept as plain UUID columns
-- (no FK) because those tables are introduced by later migrations not present
-- here. The pillar_type enum is created by 000002_content.

BEGIN;

-- Enums ---------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'mock_type') THEN
        CREATE TYPE mock_type AS ENUM (
            'coding',
            'system_design',
            'lld',
            'behavioral',
            'backend_engineering'
        );
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'mock_outcome') THEN
        CREATE TYPE mock_outcome AS ENUM (
            'strong_hire',
            'hire',
            'lean_hire',
            'no_hire',
            'strong_no_hire',
            'not_rated'
        );
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'finding_severity') THEN
        CREATE TYPE finding_severity AS ENUM (
            'info',
            'minor',
            'major',
            'blocker'
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

-- mock_interviews -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS mock_interviews (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type              mock_type NOT NULL,
    topic_id          UUID NULL REFERENCES topics (id) ON DELETE SET NULL,
    design_problem_id UUID NULL,
    company_id        UUID NULL REFERENCES companies (id) ON DELETE SET NULL,
    scheduled_at      TIMESTAMPTZ NULL,
    conducted_at      TIMESTAMPTZ NULL,
    duration_minutes  INTEGER NULL,
    outcome           mock_outcome NOT NULL DEFAULT 'not_rated',
    overall_score     NUMERIC(5,2) NULL,
    interviewer       TEXT NULL,
    transcript_md     TEXT NULL,
    summary           TEXT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_mock_user_type
    ON mock_interviews (user_id, type);
CREATE INDEX IF NOT EXISTS idx_mock_conducted
    ON mock_interviews (user_id, conducted_at);

DROP TRIGGER IF EXISTS trg_mock_interviews_updated_at ON mock_interviews;
CREATE TRIGGER trg_mock_interviews_updated_at
    BEFORE UPDATE ON mock_interviews
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- mock_findings -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mock_findings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mock_interview_id   UUID NOT NULL REFERENCES mock_interviews (id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    pillar_type         pillar_type NULL,
    topic_id            UUID NULL REFERENCES topics (id) ON DELETE SET NULL,
    severity            finding_severity NOT NULL DEFAULT 'minor',
    category            TEXT NOT NULL,
    detail              TEXT NOT NULL,
    remediation_task_id UUID NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_finding_mock
    ON mock_findings (mock_interview_id);
CREATE INDEX IF NOT EXISTS idx_finding_user_sev
    ON mock_findings (user_id, severity);

DROP TRIGGER IF EXISTS trg_mock_findings_updated_at ON mock_findings;
CREATE TRIGGER trg_mock_findings_updated_at
    BEFORE UPDATE ON mock_findings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
