-- 000007_design_problems.down.sql
-- Reverses 000007_design_problems.up.sql. Drops the design_problems table.
-- Does not drop the difficulty enum (owned by 000002) or the backend-sde3
-- track row (shared content, owned by the seeder / 000002).

BEGIN;

DROP TABLE IF EXISTS design_problems;

COMMIT;
