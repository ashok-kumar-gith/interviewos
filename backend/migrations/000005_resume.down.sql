-- 000005_resume.down.sql
-- Reverses 000005_resume.up.sql.

BEGIN;

DROP TABLE IF EXISTS resume_projects;
DROP TABLE IF EXISTS resume_profiles;

COMMIT;
