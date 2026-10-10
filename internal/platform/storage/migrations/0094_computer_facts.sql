-- A computer's facts (ADR 0146): one JSON snapshot its owner reads whole, never queried by field; facts_at is when
-- they last changed, NULL until the computer's runner first reports. No backfill: facts arrive with the runner.
ALTER TABLE pairing_computers ADD COLUMN facts TEXT NOT NULL DEFAULT '{}';
ALTER TABLE pairing_computers ADD COLUMN facts_at INTEGER;
