-- 000014_ai_invocations.down.sql
BEGIN;

DROP TABLE IF EXISTS ai_invocations;
DROP TYPE IF EXISTS ai_invocation_status;
DROP TYPE IF EXISTS ai_feature;

COMMIT;
