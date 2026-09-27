-- The setup code unlocks first run while no user exists; only its hash is kept, and a boot replaces it.
CREATE TABLE IF NOT EXISTS setup_codes (
    code_hash  TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
