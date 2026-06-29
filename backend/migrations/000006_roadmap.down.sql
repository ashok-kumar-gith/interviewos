-- 000006_roadmap.down.sql
-- Reverses 000006_roadmap.up.sql. Drops the plan tables (children first) and the
-- enums introduced by this migration.

BEGIN;

DROP TRIGGER IF EXISTS trg_plan_tasks_updated_at ON plan_tasks;
DROP TRIGGER IF EXISTS trg_plan_days_updated_at ON plan_days;
DROP TRIGGER IF EXISTS trg_roadmap_weeks_updated_at ON roadmap_weeks;
DROP TRIGGER IF EXISTS trg_roadmaps_updated_at ON roadmaps;

DROP TABLE IF EXISTS plan_tasks;
DROP TABLE IF EXISTS plan_days;
DROP TABLE IF EXISTS roadmap_weeks;
DROP TABLE IF EXISTS roadmaps;

DROP TYPE IF EXISTS plan_item_type;
DROP TYPE IF EXISTS task_status;
DROP TYPE IF EXISTS task_kind;

COMMIT;
