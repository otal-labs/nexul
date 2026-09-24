ALTER TABLE pairing_computers ADD COLUMN setup_confirmed_at INTEGER;

-- The primary key leads with computer_id, so it also serves ListPairingProviderSetups.
CREATE TABLE IF NOT EXISTS pairing_provider_setups (
    computer_id  TEXT NOT NULL REFERENCES pairing_computers(id) ON DELETE CASCADE,
    provider     TEXT NOT NULL,
    confirmed_at INTEGER,
    skills_json  TEXT NOT NULL DEFAULT '[]',
    updated_at   INTEGER NOT NULL,
    PRIMARY KEY (computer_id, provider)
);
