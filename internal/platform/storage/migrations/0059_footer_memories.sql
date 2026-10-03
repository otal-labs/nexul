-- A footer memory is named last in a play run, to conclude it; the agent now picks the ticket's column, so the run dialog's choice goes (ADR 0111).
ALTER TABLE memories ADD COLUMN footer INTEGER NOT NULL DEFAULT 0;
ALTER TABLE play_trails DROP COLUMN move_to_status_id;
