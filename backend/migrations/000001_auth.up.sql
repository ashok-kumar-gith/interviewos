-- 000001_auth.up.sql
-- Authentication schema: enums and the four auth tables (users, oauth_accounts,
-- refresh_tokens, password_reset_tokens) per docs/04-DATABASE-SCHEMA.md §3/§4.1.
--
-- Note on extensions: the schema doc specifies pgcrypto (gen_random_uuid) and
-- citext (case-insensitive email). PostgreSQL 13+ ships gen_random_uuid() in
-- core, so pgcrypto is not required. Where the citext extension is unavailable,
-- emails are stored as lowercased TEXT and case-insensitive uniqueness is
-- enforced via a functional UNIQUE index on lower(email); the application layer
-- normalizes emails to lowercase before persisting. This preserves the same
-- semantics (case-insensitive unique active email) without the extension.

BEGIN;

-- Enums ---------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'auth_provider') THEN
        CREATE TYPE auth_provider AS ENUM ('google', 'github', 'email');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('user', 'admin');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'account_status') THEN
        CREATE TYPE account_status AS ENUM ('active', 'suspended', 'deleted');
    END IF;
END$$;

-- updated_at trigger function -----------------------------------------------
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- users ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email             TEXT NOT NULL,
    email_verified_at TIMESTAMPTZ NULL,
    password_hash     TEXT NULL,
    full_name         TEXT NULL,
    avatar_url        TEXT NULL,
    role              user_role NOT NULL DEFAULT 'user',
    status            account_status NOT NULL DEFAULT 'active',
    last_login_at     TIMESTAMPTZ NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ NULL
);

-- Case-insensitive unique active email; soft-deleted rows free the address.
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email_active
    ON users (lower(email)) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- oauth_accounts ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS oauth_accounts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider         auth_provider NOT NULL,
    provider_user_id TEXT NOT NULL,
    email            TEXT NULL,
    access_token     TEXT NULL,
    refresh_token    TEXT NULL,
    expires_at       TIMESTAMPTZ NULL,
    raw_profile      JSONB NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_oauth_provider_subject
    ON oauth_accounts (provider, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_user ON oauth_accounts (user_id);

DROP TRIGGER IF EXISTS trg_oauth_accounts_updated_at ON oauth_accounts;
CREATE TRIGGER trg_oauth_accounts_updated_at
    BEFORE UPDATE ON oauth_accounts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- refresh_tokens ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL,
    family_id   UUID NOT NULL,
    user_agent  TEXT NULL,
    ip_address  INET NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ NULL,
    replaced_by UUID NULL REFERENCES refresh_tokens (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_refresh_token_hash
    ON refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_rt_user ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_rt_family ON refresh_tokens (family_id);
CREATE INDEX IF NOT EXISTS idx_rt_active
    ON refresh_tokens (user_id) WHERE revoked_at IS NULL;

DROP TRIGGER IF EXISTS trg_refresh_tokens_updated_at ON refresh_tokens;
CREATE TRIGGER trg_refresh_tokens_updated_at
    BEFORE UPDATE ON refresh_tokens
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- password_reset_tokens -----------------------------------------------------
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_prt_token_hash
    ON password_reset_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_prt_user
    ON password_reset_tokens (user_id) WHERE used_at IS NULL;

DROP TRIGGER IF EXISTS trg_password_reset_tokens_updated_at ON password_reset_tokens;
CREATE TRIGGER trg_password_reset_tokens_updated_at
    BEFORE UPDATE ON password_reset_tokens
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
