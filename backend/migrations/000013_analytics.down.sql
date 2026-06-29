-- 000013_analytics.down.sql
-- Reverses 000013_analytics.up.sql.

BEGIN;

DROP TRIGGER IF EXISTS trg_readiness_snapshots_updated_at ON readiness_snapshots;
DROP TABLE IF EXISTS readiness_snapshots;

COMMIT;
