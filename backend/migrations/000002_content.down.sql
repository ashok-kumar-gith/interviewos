-- 000002_content.down.sql
-- Reverses 000002_content.up.sql. Drops content tables in reverse FK order and
-- the content enum types. Does not drop set_updated_at() (owned by 000001).

BEGIN;

DROP TABLE IF EXISTS problem_company_frequency;
DROP TABLE IF EXISTS company_weights;
DROP TABLE IF EXISTS companies;
DROP TABLE IF EXISTS problem_sources;
DROP TABLE IF EXISTS problem_patterns;
DROP TABLE IF EXISTS problems;
DROP TABLE IF EXISTS patterns;
DROP TABLE IF EXISTS topic_resources;
DROP TABLE IF EXISTS resources;
DROP TABLE IF EXISTS subtopics;
DROP TABLE IF EXISTS topics;
DROP TABLE IF EXISTS pillars;
DROP TABLE IF EXISTS tracks;

DROP TYPE IF EXISTS problem_platform;
DROP TYPE IF EXISTS problem_source_name;
DROP TYPE IF EXISTS priority;
DROP TYPE IF EXISTS difficulty;
DROP TYPE IF EXISTS resource_type;
DROP TYPE IF EXISTS pillar_type;

COMMIT;
