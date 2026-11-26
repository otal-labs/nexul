-- The labels a computer's owner sets on it, a JSON array of strings; a play's needs match them (ADR 0197).
ALTER TABLE pairing_computers ADD COLUMN labels TEXT NOT NULL DEFAULT '[]';
