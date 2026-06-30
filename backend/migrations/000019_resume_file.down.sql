-- 000019_resume_file.down.sql
-- Reverses 000019_resume_file.up.sql.

BEGIN;

DROP TRIGGER IF EXISTS trg_resume_files_updated_at ON resume_files;
DROP TABLE IF EXISTS resume_files;

COMMIT;
