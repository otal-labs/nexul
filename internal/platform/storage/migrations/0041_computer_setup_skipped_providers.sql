-- Driver kinds the owner left out of this computer's setup runs; the default empty list keeps every provider in, as before.
ALTER TABLE pairing_computers ADD COLUMN setup_skipped_providers TEXT NOT NULL DEFAULT '[]';
