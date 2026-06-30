-- 000019_resume_file.up.sql
-- Stores the user's uploaded resume file (PDF or DOCX) bytes directly in
-- Postgres (no object store locally). One current file per user, enforced by a
-- partial unique index on user_id WHERE deleted_at IS NULL — replacing a resume
-- soft-deletes the old row and inserts the new one. user_id keeps the hard FK to
-- users(id) for the user-owned cascade.

BEGIN;

CREATE TABLE IF NOT EXISTS resume_files (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    file_name     TEXT NOT NULL,
    content_type  TEXT NOT NULL,
    size_bytes    INTEGER NOT NULL,
    content       BYTEA NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ NULL
);

-- One live resume file per user.
CREATE UNIQUE INDEX IF NOT EXISTS uq_resume_files_user
    ON resume_files (user_id) WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_resume_files_updated_at ON resume_files;
CREATE TRIGGER trg_resume_files_updated_at
    BEFORE UPDATE ON resume_files
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
