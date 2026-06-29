-- 000001_auth.down.sql
-- Reverse the auth schema. Drops tables, the shared trigger function, and the
-- auth enums. Extensions (pgcrypto, citext) are left installed since they are
-- harmless and shared with later migrations.

BEGIN;

DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS oauth_accounts;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS set_updated_at();

DROP TYPE IF EXISTS account_status;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS auth_provider;

COMMIT;
