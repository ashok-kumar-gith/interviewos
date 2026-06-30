-- 000020_problem_solution.up.sql
-- DSA solution capture: extend user_problem_progress (000011) so a user can
-- store the solution they wrote for a problem — the code, its language, and a
-- short approach note — alongside the existing solved/solved_at tracking. This
-- powers "which problem is solved, when, and with what solution".
--
-- These columns are additive and nullable, so existing rows and the task-
-- completion upsert path are unaffected.

BEGIN;

ALTER TABLE user_problem_progress
    ADD COLUMN IF NOT EXISTS solution_code     TEXT NULL,
    ADD COLUMN IF NOT EXISTS solution_language TEXT NULL,
    ADD COLUMN IF NOT EXISTS solution_notes    TEXT NULL,
    ADD COLUMN IF NOT EXISTS solution_updated_at TIMESTAMPTZ NULL;

COMMIT;
