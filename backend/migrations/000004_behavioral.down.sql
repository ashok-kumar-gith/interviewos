-- 000004_behavioral.down.sql
-- Reverse the behavioral schema: drop the behavioral_stories table and the
-- story_theme enum. The shared set_updated_at() function is owned by migration
-- 000001 and is left intact.

BEGIN;

DROP TABLE IF EXISTS behavioral_stories;

DROP TYPE IF EXISTS story_theme;

COMMIT;
