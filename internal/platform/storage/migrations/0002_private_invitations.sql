ALTER TABLE users ADD COLUMN account_status TEXT NOT NULL DEFAULT 'active' CHECK (account_status IN ('active', 'disabled', 'removed'));

CREATE TABLE IF NOT EXISTS invitations (
    id          TEXT PRIMARY KEY,
    token_hash  TEXT NOT NULL UNIQUE,
    invited_by  TEXT NOT NULL REFERENCES users(id),
    created_at  INTEGER NOT NULL,
    expires_at  INTEGER NOT NULL,
    redeemed_at INTEGER,
    redeemed_by TEXT REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS invitation_grants (
    invitation_id TEXT NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    workspace_id  TEXT NOT NULL,
    role_id       TEXT NOT NULL,
    allow_json    TEXT NOT NULL DEFAULT '[]',
    deny_json     TEXT NOT NULL DEFAULT '[]',
    PRIMARY KEY (invitation_id, workspace_id)
);

CREATE TABLE IF NOT EXISTS invitation_oauth_handoffs (
    id                TEXT PRIMARY KEY,
    invitation_id     TEXT NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    oauth_state_hash  TEXT UNIQUE,
    acceptance_hash   TEXT UNIQUE,
    provider          TEXT NOT NULL,
    provider_user_id  TEXT,
    login             TEXT,
    name              TEXT,
    avatar_url        TEXT,
    existing_user_id  TEXT REFERENCES users(id),
    admitted_user_id  TEXT REFERENCES users(id),
    completed_at      INTEGER,
    created_at        INTEGER NOT NULL,
    expires_at        INTEGER NOT NULL
);
