-- A passkey signs its owner in on the one domain (rp_id) it was made for; credential is the verifier's record as JSON
-- (public key, sign count, flags), rewritten after each sign-in.
CREATE TABLE IF NOT EXISTS user_passkeys (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    rp_id        TEXT NOT NULL,
    credential   TEXT NOT NULL,
    created_at   INTEGER NOT NULL,
    last_used_at INTEGER
);
-- Serves ListPasskeysByUser (oldest first) and the user-scoped rename and delete.
CREATE INDEX IF NOT EXISTS idx_user_passkeys_user ON user_passkeys(user_id, created_at, id);
