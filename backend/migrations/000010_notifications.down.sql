-- 000010_notifications.down.sql
-- Reverse the notifications schema: drop the notifications table and the three
-- notification_* enums. The shared set_updated_at() function is owned by
-- migration 000001 and is left intact.

BEGIN;

DROP TABLE IF EXISTS notifications;

DROP TYPE IF EXISTS notification_status;
DROP TYPE IF EXISTS notification_channel;
DROP TYPE IF EXISTS notification_type;

COMMIT;
