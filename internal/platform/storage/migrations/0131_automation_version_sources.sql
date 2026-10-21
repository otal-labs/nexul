-- The source file a push came from, beside the bundle the host runs; empty for versions pushed before it was kept.
ALTER TABLE automation_versions ADD COLUMN source TEXT NOT NULL DEFAULT '';
