-- Automations run on named automations hosts that enroll for their own credential; the tokens file is gone.
CREATE TABLE IF NOT EXISTS automation_hosts (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    machine    TEXT NOT NULL DEFAULT '',
    os         TEXT NOT NULL DEFAULT '',
    arch       TEXT NOT NULL DEFAULT '',
    version    TEXT NOT NULL DEFAULT '',
    last_seen  INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS automation_host_enrollment_codes (
    code_hash  TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    machine    TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);

-- A removed host's credential stays as a tombstone (revoked_at set) so a returning host is told it was removed.
CREATE TABLE IF NOT EXISTS automation_host_credentials (
    credential_hash TEXT PRIMARY KEY,
    host_id         TEXT NOT NULL,
    host_name       TEXT NOT NULL,
    created_at      INTEGER NOT NULL,
    revoked_at      INTEGER
);
-- Serves revoking a host's credentials on removal and finding its live one for automation token checks.
CREATE INDEX IF NOT EXISTS idx_automation_host_credentials_host ON automation_host_credentials(host_id, revoked_at);

-- NULL places an automation on the bundled host named instance; removing a host moves its automations back there.
ALTER TABLE automations ADD COLUMN host_id TEXT REFERENCES automation_hosts(id) ON DELETE SET NULL;
