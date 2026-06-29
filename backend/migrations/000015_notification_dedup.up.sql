-- 000015_notification_dedup.up.sql
-- Adds a dedup_key to notifications so the generator (internal/notification)
-- can upsert digest-style notifications idempotently: re-running the daily
-- generation for the same user/day must NOT create duplicate rows.
--
-- dedup_key is a stable, generator-chosen string (e.g. "today_plan:2026-06-29").
-- It is nullable: notifications enqueued ad hoc by other modules (via the
-- Notifier) leave it NULL and are never deduped. The partial unique index only
-- constrains rows that set a dedup_key and are not soft-deleted, so a user can
-- still have many NULL-keyed notifications.

BEGIN;

ALTER TABLE notifications ADD COLUMN IF NOT EXISTS dedup_key TEXT NULL;

-- One live (non-deleted) notification per (user, dedup_key). The generator
-- relies on this to make daily generation idempotent.
CREATE UNIQUE INDEX IF NOT EXISTS uq_notif_user_dedup
    ON notifications (user_id, dedup_key)
    WHERE dedup_key IS NOT NULL AND deleted_at IS NULL;

COMMIT;
