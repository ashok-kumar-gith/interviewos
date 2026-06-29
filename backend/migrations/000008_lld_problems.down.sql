-- 000008_lld_problems.down.sql
-- Reverses 000008_lld_problems.up.sql. Drops the lld_problems table (and its
-- seeded rows, indexes, and trigger). The track/pillar upserts performed by the
-- up migration are intentionally left in place: they are shared content owned by
-- the content seeder and earlier migrations, not by this one.

BEGIN;

DROP TRIGGER IF EXISTS trg_lld_problems_updated_at ON lld_problems;
DROP TABLE IF EXISTS lld_problems;

COMMIT;
