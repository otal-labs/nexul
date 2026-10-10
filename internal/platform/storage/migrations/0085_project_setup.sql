-- A project's setup record (ADR 0143); every project made before it counts as finished, so no sidebar changes.
ALTER TABLE projects ADD COLUMN setup_finished INTEGER NOT NULL DEFAULT 1;
ALTER TABLE projects ADD COLUMN setup_steps TEXT NOT NULL DEFAULT '{}';
