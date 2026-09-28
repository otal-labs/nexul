-- A user signs in through any of several provider accounts (ADR 0040, amended); the users row loses its provider columns.
CREATE TABLE IF NOT EXISTS user_identities (
    user_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    login            TEXT NOT NULL,
    name             TEXT NOT NULL DEFAULT '',
    avatar_url       TEXT NOT NULL DEFAULT '',
    created_at       INTEGER NOT NULL,
    PRIMARY KEY (provider, provider_user_id),
    -- One account per provider per user; the unique index also serves ListIdentitiesByUser and CountIdentitiesByUser.
    UNIQUE (user_id, provider)
);
INSERT INTO user_identities (user_id, provider, provider_user_id, login, name, avatar_url, created_at)
SELECT id, provider, provider_user_id, login, name, avatar_url, created_at FROM users;

CREATE TABLE users_new (
    id                   TEXT PRIMARY KEY,
    login                TEXT NOT NULL,
    name                 TEXT NOT NULL DEFAULT '',
    avatar_url           TEXT NOT NULL DEFAULT '',
    first_login_done     INTEGER NOT NULL DEFAULT 0,
    can_create_workspace INTEGER NOT NULL DEFAULT 0,
    display_name         TEXT,
    avatar_override_url  TEXT,
    account_status       TEXT NOT NULL DEFAULT 'active' CHECK (account_status IN ('active', 'disabled', 'removed')),
    created_at           INTEGER NOT NULL,
    updated_at           INTEGER NOT NULL
);
INSERT INTO users_new (id, login, name, avatar_url, first_login_done, can_create_workspace, display_name, avatar_override_url, account_status, created_at, updated_at)
SELECT id, login, name, avatar_url, first_login_done, can_create_workspace, display_name, avatar_override_url, account_status, created_at, updated_at
FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
-- Serves GetUserByLogin and ListUsers.
CREATE INDEX IF NOT EXISTS idx_users_login ON users(login);
