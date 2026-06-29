-- 000010_notifications.up.sql
-- Notifications module schema: the notification_type / notification_channel /
-- notification_status enums and the notifications table per
-- docs/04-DATABASE-SCHEMA.md §3/§4.3 and docs/openapi.yaml (Notification schema).
--
-- notifications is a user-data table: it carries deleted_at (soft delete) and
-- FK→users(id) ON DELETE CASCADE. The shared set_updated_at() trigger function
-- is created by migration 000001; we (re)create it defensively so this migration
-- also applies cleanly in isolation against a fresh database.
--
-- At GA only the in_app channel is delivered (per ADR open-question default);
-- the email/push enum values exist so other modules can enqueue without a
-- schema change when those channels ship.

BEGIN;

-- Enums ---------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'notification_type') THEN
        CREATE TYPE notification_type AS ENUM (
            'today_plan',
            'revision_due',
            'weekly_review',
            'missed_goal',
            'streak_reminder',
            'readiness_milestone',
            'mock_scheduled',
            'system'
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'notification_channel') THEN
        CREATE TYPE notification_channel AS ENUM (
            'in_app',
            'email',
            'push'
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'notification_status') THEN
        CREATE TYPE notification_status AS ENUM (
            'unread',
            'read',
            'dismissed'
        );
    END IF;
END$$;

-- Defensive: ensure the shared updated_at trigger function exists even if this
-- migration is applied in isolation against a fresh database.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- notifications -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type        notification_type NOT NULL,
    channel     notification_channel NOT NULL DEFAULT 'in_app',
    status      notification_status NOT NULL DEFAULT 'unread',
    title       TEXT NOT NULL,
    body        TEXT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    read_at     TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ NULL
);

-- Status-filtered listing per user (GET /notifications?status=...).
CREATE INDEX IF NOT EXISTS idx_notif_user_status
    ON notifications (user_id, status);
-- Newest-first listing per user (default sort).
CREATE INDEX IF NOT EXISTS idx_notif_user_created
    ON notifications (user_id, created_at DESC);
-- Hot path: the unread badge / mark-all query.
CREATE INDEX IF NOT EXISTS idx_notif_unread
    ON notifications (user_id) WHERE status = 'unread';

DROP TRIGGER IF EXISTS trg_notifications_updated_at ON notifications;
CREATE TRIGGER trg_notifications_updated_at
    BEFORE UPDATE ON notifications
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
