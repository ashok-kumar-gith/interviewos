-- 000016_audit_logs.up.sql
-- Security audit trail (FR-AUDIT-003 / docs/04-DATABASE-SCHEMA.md §4 "audit_logs").
-- Append-only log of security-relevant events: login success/failure, register,
-- password reset, logout, refresh-token reuse detection, and account deletion.
-- Written application-side (best-effort, never blocking the request) by the auth
-- service. No updated_at/deleted_at: rows are immutable once written.
--
-- user_id is nullable (FK ... ON DELETE SET NULL): a failed login or a reset for
-- a non-existent account has no actor, and deleting a user must not erase their
-- audit history. metadata is a free-form JSONB bag for event-specific context
-- (e.g. {"reason":"invalid_credentials"}).

BEGIN;

CREATE TABLE IF NOT EXISTS audit_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    action     TEXT NOT NULL,
    ip_address INET NULL,
    user_agent TEXT NULL,
    metadata   JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_action_time ON audit_logs (action, created_at DESC);

COMMIT;
