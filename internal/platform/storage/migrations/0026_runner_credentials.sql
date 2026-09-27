-- Runners authenticate with their own credential, traded once for an enrollment code; the shared secret is gone.
-- Runners registered through the shared secret hold no credential and can never connect again.
ALTER TABLE instance_settings DROP COLUMN runner_secret;
DELETE FROM runners;

-- Serves enrollment's name check and upgrade dispatch to the runner named instance.
CREATE UNIQUE INDEX IF NOT EXISTS idx_runners_name ON runners(name);

CREATE TABLE IF NOT EXISTS runner_enrollment_codes (
    code_hash  TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    machine    TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);

-- A removed runner's credential stays as a tombstone (revoked_at set) so a returning runner is told it was removed.
CREATE TABLE IF NOT EXISTS runner_credentials (
    credential_hash TEXT PRIMARY KEY,
    runner_id       TEXT NOT NULL,
    runner_name     TEXT NOT NULL,
    created_at      INTEGER NOT NULL,
    revoked_at      INTEGER
);
-- Serves revoking a runner's credentials on removal.
CREATE INDEX IF NOT EXISTS idx_runner_credentials_runner ON runner_credentials(runner_id);
