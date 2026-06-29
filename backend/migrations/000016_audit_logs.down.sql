-- 000016_audit_logs.down.sql
BEGIN;

DROP INDEX IF EXISTS idx_audit_action_time;
DROP INDEX IF EXISTS idx_audit_user;
DROP TABLE IF EXISTS audit_logs;

COMMIT;
