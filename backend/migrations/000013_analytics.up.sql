-- 000013_analytics.up.sql
-- Analytics Engine persistence per docs/04-DATABASE-SCHEMA.md (readiness_snapshots)
-- and SRS §6.2. The daily readiness snapshot powers the readiness-over-time chart
-- (GET /analytics/snapshots) and the Estimated Interview Readiness Date projection.
--
-- Self-contained references (mirroring 000011_progress): user_id keeps a hard FK
-- to users(id) so the user-owned cascade holds; roadmap_id is stored as a plain
-- UUID WITHOUT a hard FK to keep this migration independently applicable (the
-- roadmaps table lives in 000006_roadmap). Integrity is validated at the service
-- layer, consistent with the cross-module convention used elsewhere.

BEGIN;

-- readiness_snapshots: one daily readiness rollup per user.
CREATE TABLE IF NOT EXISTS readiness_snapshots (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    roadmap_id            UUID NULL,
    snapshot_date         DATE NOT NULL,
    overall_readiness     NUMERIC(5,2) NOT NULL DEFAULT 0,
    pillar_readiness      JSONB NOT NULL DEFAULT '{}',
    completion_pct        NUMERIC(5,2) NOT NULL DEFAULT 0,
    avg_confidence        NUMERIC(4,2) NULL,
    revision_health       NUMERIC(5,2) NULL,
    estimated_ready_date  DATE NULL,
    weak_topics           JSONB NOT NULL DEFAULT '[]',
    strong_topics         JSONB NOT NULL DEFAULT '[]',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at            TIMESTAMPTZ NULL
);

-- One live snapshot per (user, day): the daily snapshot is idempotent (upsert),
-- per NFR-REL-006.
CREATE UNIQUE INDEX IF NOT EXISTS uq_snap_user_date
    ON readiness_snapshots (user_id, snapshot_date) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_snap_user_date
    ON readiness_snapshots (user_id, snapshot_date DESC);

-- Maintain updated_at (set_updated_at() created by migration 000001_auth).
DROP TRIGGER IF EXISTS trg_readiness_snapshots_updated_at ON readiness_snapshots;
CREATE TRIGGER trg_readiness_snapshots_updated_at
    BEFORE UPDATE ON readiness_snapshots
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
