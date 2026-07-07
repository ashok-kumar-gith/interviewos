-- Add code-review and LLD-review AI features to the ai_feature enum so
-- ai_invocations can record these new review calls. ADD VALUE IF NOT EXISTS is
-- idempotent and, on PostgreSQL 12+, is permitted inside the migration
-- transaction because the new values are not used within the same transaction.
ALTER TYPE ai_feature ADD VALUE IF NOT EXISTS 'code_review';
ALTER TYPE ai_feature ADD VALUE IF NOT EXISTS 'lld_review';
