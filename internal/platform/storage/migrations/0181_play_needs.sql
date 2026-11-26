-- The labels a computer must have to take a play's runs, a JSON array; empty needs nothing (ADR 0197).
ALTER TABLE plays ADD COLUMN needs TEXT NOT NULL DEFAULT '[]';
