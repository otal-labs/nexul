-- The computer's MCP token as the setup turns wrote it into its providers' configs, encrypted; a re-run reuses it.
ALTER TABLE pairing_computers ADD COLUMN setup_mcp_token TEXT NOT NULL DEFAULT '';

-- One provider's setup turn on a computer, with its transcript saved redacted.
CREATE TABLE IF NOT EXISTS pairing_setup_turns (
    id            TEXT PRIMARY KEY,
    run_id        TEXT NOT NULL,
    computer_id   TEXT NOT NULL REFERENCES pairing_computers(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    state         TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT '',
    transcript    TEXT NOT NULL DEFAULT '[]',
    started_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    ended_at      INTEGER
);

-- Serves the ON DELETE CASCADE from pairing_computers and ListPairingSetupTurnsLatest's newest turn per provider.
CREATE INDEX IF NOT EXISTS idx_pairing_setup_turns_computer ON pairing_setup_turns(computer_id, provider, started_at, id);
