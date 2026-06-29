-- 000011_progress.down.sql
-- Reverses 000011_progress.up.sql.

BEGIN;

DROP TABLE IF EXISTS streak_days;
DROP TABLE IF EXISTS study_sessions;
DROP TABLE IF EXISTS user_problem_progress;
DROP TABLE IF EXISTS user_topic_progress;

DROP TYPE IF EXISTS progress_status;

COMMIT;
