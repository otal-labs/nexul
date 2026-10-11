-- Where a queued run's person picked it should run, when several computers are paired and none is the default: the
-- computer and the T3 project on it. Empty until picked; the pick changes neither their defaults nor their links.
ALTER TABLE play_queue ADD COLUMN computer_id TEXT NOT NULL DEFAULT '';
ALTER TABLE play_queue ADD COLUMN harness_project_id TEXT NOT NULL DEFAULT '';
