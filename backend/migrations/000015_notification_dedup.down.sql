-- 000015_notification_dedup.down.sql

BEGIN;

DROP INDEX IF EXISTS uq_notif_user_dedup;
ALTER TABLE notifications DROP COLUMN IF EXISTS dedup_key;

COMMIT;
