-- A connect code signs one phone in once (ADR 0083); only its hash is kept, and a used or replaced code is deleted.
CREATE TABLE IF NOT EXISTS connect_codes (
    code_hash  TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
-- Serves ReplaceConnectCode's delete of the user's earlier code.
CREATE INDEX IF NOT EXISTS idx_connect_codes_user ON connect_codes(user_id);
