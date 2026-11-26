-- A computer its owner switched off for plays takes no run, pressed, queued or @Agent, and keeps commands and logs (ADR 0211).
ALTER TABLE pairing_computers ADD COLUMN plays_off INTEGER NOT NULL DEFAULT 0;
