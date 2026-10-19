-- How many runs a person may have going before their next auto run in this workspace waits; one, as before, for every
-- workspace that exists.
ALTER TABLE workspaces ADD COLUMN auto_play_runs_at_once INTEGER NOT NULL DEFAULT 1;
