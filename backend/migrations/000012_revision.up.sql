-- 000012_revision.up.sql
-- Revision Engine (spaced repetition) persistence per docs/04-DATABASE-SCHEMA.md
-- §revision_items and docs/02-SRS.md §6.1. One active revision item per
-- (user, item_type, item_id); the interval ladder [1,3,7,15,30] days is applied
-- in the service layer (internal/revision). `ease` is reserved/inert at GA
-- (stored, never mutated).
--
-- Consistent with 000011_progress: item_id references a content/roadmap item
-- polymorphically and is stored as a plain UUID WITHOUT a hard FK (it can point
-- to topic|problem|design_problem|lld_problem); user_id keeps its hard FK so the
-- user-owned cascade holds.

BEGIN;

-- recall_result enum (introduced here; absent in 000001-000011).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'recall_result') THEN
        CREATE TYPE recall_result AS ENUM ('correct','incorrect');
    END IF;
END
$$;

-- revision_items: spaced-repetition state (1/3/7/15/30-day ladder).
CREATE TABLE IF NOT EXISTS revision_items (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    item_type         plan_item_type NOT NULL,
    item_id           UUID NOT NULL,
    pillar_type       pillar_type NOT NULL,
    interval_days     SMALLINT NOT NULL DEFAULT 1,
    stage             SMALLINT NOT NULL DEFAULT 0,
    ease              NUMERIC(4,2) NOT NULL DEFAULT 2.50,
    due_at            DATE NOT NULL,
    last_reviewed_at  TIMESTAMPTZ NULL,
    last_recall       recall_result NULL,
    review_count      INTEGER NOT NULL DEFAULT 0,
    lapse_count       INTEGER NOT NULL DEFAULT 0,
    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ NULL
);

-- At most one active (non-deleted) revision item per (user, item_type, item_id)
-- — dedupe per FR-REV-007.
CREATE UNIQUE INDEX IF NOT EXISTS uq_rev_user_item
    ON revision_items (user_id, item_type, item_id) WHERE deleted_at IS NULL;
-- Due lookup for active items.
CREATE INDEX IF NOT EXISTS idx_rev_due
    ON revision_items (user_id, due_at) WHERE is_active;
-- Polymorphic reverse lookup.
CREATE INDEX IF NOT EXISTS idx_rev_poly ON revision_items (item_type, item_id);

-- Maintain updated_at (set_updated_at() created by migration 000001_auth).
DROP TRIGGER IF EXISTS trg_revision_items_updated_at ON revision_items;
CREATE TRIGGER trg_revision_items_updated_at
    BEFORE UPDATE ON revision_items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
