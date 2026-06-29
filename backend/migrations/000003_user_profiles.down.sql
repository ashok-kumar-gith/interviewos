-- 000003_user_profiles.down.sql
-- Reverses 000003_user_profiles.up.sql.

BEGIN;

DROP TRIGGER IF EXISTS trg_user_profiles_updated_at ON user_profiles;
DROP TABLE IF EXISTS user_profiles;

COMMIT;
