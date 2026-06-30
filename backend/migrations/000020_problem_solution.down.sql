-- 000020_problem_solution.down.sql

BEGIN;

ALTER TABLE user_problem_progress
    DROP COLUMN IF EXISTS solution_code,
    DROP COLUMN IF EXISTS solution_language,
    DROP COLUMN IF EXISTS solution_notes,
    DROP COLUMN IF EXISTS solution_updated_at;

COMMIT;
