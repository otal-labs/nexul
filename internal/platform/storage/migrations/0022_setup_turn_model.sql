-- The model a setup turn ran on, as the user picked it; empty means the provider's own default.
ALTER TABLE pairing_setup_turns ADD COLUMN model TEXT NOT NULL DEFAULT '';
