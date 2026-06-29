-- 000006_roadmap.up.sql
-- Roadmap / plan persistence per docs/04-DATABASE-SCHEMA.md §4.3:
--   roadmaps, roadmap_weeks, plan_days, plan_tasks.
-- These tables are the durable output of the Curriculum Engine (internal/curriculum)
-- and the read model for the daily "Today" view.
--
-- Self-contained references (mirroring 000003_user_profiles): track_id,
-- profile_id and target_company_id reference tables that live in other
-- migrations (tracks/user_profiles/companies). To keep this migration
-- independently applicable they are stored as plain UUIDs WITHOUT hard FKs;
-- integrity is validated at the application/service layer. user_id keeps its
-- hard FK to users(id) (created by 000001_auth) so the user-owned cascade holds.
--
-- The polymorphic (item_type, item_id) on plan_tasks intentionally has no DB FK
-- (it can point at topic/problem/resource/... ); integrity is enforced by the
-- service layer (04-DATABASE-SCHEMA.md §4.3 note).

BEGIN;

-- Enums introduced for the plan model (created here; absent in 000001-000005).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'task_kind') THEN
        CREATE TYPE task_kind AS ENUM ('study','solve','read','watch','revise','mock');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'task_status') THEN
        CREATE TYPE task_status AS ENUM ('pending','in_progress','completed','skipped','rescheduled');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'plan_item_type') THEN
        CREATE TYPE plan_item_type AS ENUM ('topic','subtopic','problem','resource','design_problem','lld_problem','behavioral_story','revision_item');
    END IF;
END
$$;

-- roadmaps: one active roadmap per user (history retained via is_active/deleted_at).
CREATE TABLE IF NOT EXISTS roadmaps (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    track_id           UUID NOT NULL,
    profile_id         UUID NOT NULL,
    target_company_id  UUID NULL,
    start_date         DATE NOT NULL,
    end_date           DATE NOT NULL,
    total_weeks        SMALLINT NOT NULL DEFAULT 12 CHECK (total_weeks BETWEEN 1 AND 52),
    hours_per_week     SMALLINT NOT NULL CHECK (hours_per_week BETWEEN 1 AND 80),
    status             TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','completed','archived')),
    is_active          BOOLEAN NOT NULL DEFAULT true,
    generation_params  JSONB NOT NULL DEFAULT '{}',
    generated_by       TEXT NOT NULL DEFAULT 'engine' CHECK (generated_by IN ('engine','ai')),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_roadmap_user ON roadmaps (user_id);
-- Exactly one active, non-deleted roadmap per user.
CREATE UNIQUE INDEX IF NOT EXISTS uq_roadmap_user_active
    ON roadmaps (user_id) WHERE is_active AND deleted_at IS NULL;

-- roadmap_weeks: 1..total_weeks per roadmap.
CREATE TABLE IF NOT EXISTS roadmap_weeks (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    roadmap_id    UUID NOT NULL REFERENCES roadmaps (id) ON DELETE CASCADE,
    week_number   SMALLINT NOT NULL CHECK (week_number >= 1),
    start_date    DATE NOT NULL,
    end_date      DATE NOT NULL,
    theme         TEXT NULL,
    focus_pillars JSONB NOT NULL DEFAULT '[]',
    planned_hours NUMERIC(6,2) NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_week_roadmap_number
    ON roadmap_weeks (roadmap_id, week_number);
CREATE INDEX IF NOT EXISTS idx_week_roadmap ON roadmap_weeks (roadmap_id);

-- plan_days: one dated day per week. user_id denormalized for fast scoping.
CREATE TABLE IF NOT EXISTS plan_days (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    roadmap_week_id   UUID NOT NULL REFERENCES roadmap_weeks (id) ON DELETE CASCADE,
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    date              DATE NOT NULL,
    planned_minutes   INTEGER NOT NULL DEFAULT 0,
    completed_minutes INTEGER NOT NULL DEFAULT 0,
    is_rest_day       BOOLEAN NOT NULL DEFAULT false,
    summary           TEXT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_planday_week_date
    ON plan_days (roadmap_week_id, date);
-- Hot path: fetch a user's plan-day by date.
CREATE INDEX IF NOT EXISTS idx_planday_user_date ON plan_days (user_id, date);

-- plan_tasks: the unified Today list. Polymorphic content reference.
CREATE TABLE IF NOT EXISTS plan_tasks (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_day_id        UUID NOT NULL REFERENCES plan_days (id) ON DELETE CASCADE,
    user_id            UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind               task_kind NOT NULL,
    item_type          plan_item_type NOT NULL,
    item_id            UUID NOT NULL,
    pillar_type        pillar_type NOT NULL,
    title              TEXT NOT NULL,
    description        TEXT NULL,
    objectives         JSONB NOT NULL DEFAULT '[]',
    estimated_minutes  INTEGER NOT NULL DEFAULT 30,
    priority           priority NOT NULL DEFAULT 'medium',
    difficulty         difficulty NULL,
    status             task_status NOT NULL DEFAULT 'pending',
    sort_order         INTEGER NOT NULL DEFAULT 0,
    confidence         SMALLINT NULL CHECK (confidence BETWEEN 1 AND 5),
    time_spent_minutes INTEGER NULL,
    completion_notes   TEXT NULL,
    revision_item_id   UUID NULL,
    rescheduled_from   UUID NULL REFERENCES plan_tasks (id) ON DELETE SET NULL,
    completed_at       TIMESTAMPTZ NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_task_day ON plan_tasks (plan_day_id);
CREATE INDEX IF NOT EXISTS idx_task_user_status ON plan_tasks (user_id, status);
CREATE INDEX IF NOT EXISTS idx_task_poly ON plan_tasks (item_type, item_id);
CREATE INDEX IF NOT EXISTS idx_task_user_kind ON plan_tasks (user_id, kind);
CREATE INDEX IF NOT EXISTS idx_task_pending
    ON plan_tasks (user_id, plan_day_id) WHERE status = 'pending';

-- Maintain updated_at (set_updated_at() created by migration 000001_auth).
DROP TRIGGER IF EXISTS trg_roadmaps_updated_at ON roadmaps;
CREATE TRIGGER trg_roadmaps_updated_at
    BEFORE UPDATE ON roadmaps
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_roadmap_weeks_updated_at ON roadmap_weeks;
CREATE TRIGGER trg_roadmap_weeks_updated_at
    BEFORE UPDATE ON roadmap_weeks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_plan_days_updated_at ON plan_days;
CREATE TRIGGER trg_plan_days_updated_at
    BEFORE UPDATE ON plan_days
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_plan_tasks_updated_at ON plan_tasks;
CREATE TRIGGER trg_plan_tasks_updated_at
    BEFORE UPDATE ON plan_tasks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
