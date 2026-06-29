-- 000012_revision.down.sql
-- Reverses 000012_revision.up.sql.

BEGIN;

DROP TABLE IF EXISTS revision_items;

DROP TYPE IF EXISTS recall_result;

COMMIT;
