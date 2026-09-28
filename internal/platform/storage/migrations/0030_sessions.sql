-- A session is one signed-in device (ADR 0083); only the token's hash is kept, and deleting the row signs it out.
CREATE TABLE IF NOT EXISTS sessions (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash     TEXT NOT NULL UNIQUE,
    client         TEXT NOT NULL,
    platform       TEXT NOT NULL DEFAULT '',
    label          TEXT NOT NULL DEFAULT '',
    ip             TEXT NOT NULL DEFAULT '',
    created_at     INTEGER NOT NULL,
    last_active_at INTEGER NOT NULL,
    expires_at     INTEGER NOT NULL
);
-- Serves ListSessionsByUser (most recently active first) and the per-user sweeps in DeleteOtherSessions and DeleteExpiredSessions.
CREATE INDEX IF NOT EXISTS idx_sessions_user_active ON sessions(user_id, last_active_at DESC, id);
