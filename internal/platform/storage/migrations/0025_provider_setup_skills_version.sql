-- The nexul-memory version a provider's confirming setup installed; empty for confirmations made before skills were versioned.
ALTER TABLE pairing_provider_setups ADD COLUMN skills_version TEXT NOT NULL DEFAULT '';
