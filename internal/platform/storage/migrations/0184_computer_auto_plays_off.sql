-- A computer its owner switched off for auto plays takes no run an auto play starts, but still takes the ones a person presses (ADR 0213).
ALTER TABLE pairing_computers ADD COLUMN auto_plays_off INTEGER NOT NULL DEFAULT 0;
