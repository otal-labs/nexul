-- A computer's runs at once is the only limit on automatic runs (ADR 0214): the workspace's daily cap and the resumes that lifted it go.
DROP TABLE IF EXISTS play_queue_resumes;
ALTER TABLE workspaces DROP COLUMN auto_play_daily_cap;
