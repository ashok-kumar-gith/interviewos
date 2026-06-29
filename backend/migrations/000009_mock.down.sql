-- 000009_mock.down.sql
-- Reverse the mock schema: drop mock_findings and mock_interviews (child first)
-- and the mock_type, mock_outcome, finding_severity enums. The shared
-- set_updated_at() function and the pillar_type enum are owned by earlier
-- migrations and are left intact.

BEGIN;

DROP TABLE IF EXISTS mock_findings;
DROP TABLE IF EXISTS mock_interviews;

DROP TYPE IF EXISTS finding_severity;
DROP TYPE IF EXISTS mock_outcome;
DROP TYPE IF EXISTS mock_type;

COMMIT;
