-- The checkouts a person picked for a project on a computer, one per repository of the project, so a computer holding
-- both an app and a tests repository keeps a checkout of each. repository is the GitHub owner/name in lowercase; '' is a
-- pick made before, which counts for the repository it is a clone of.
CREATE TABLE IF NOT EXISTS pairing_project_checkouts (
    user_id            TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id         TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    computer_id        TEXT NOT NULL REFERENCES pairing_computers(id) ON DELETE CASCADE,
    repository         TEXT NOT NULL,
    harness_project_id TEXT NOT NULL,
    PRIMARY KEY (user_id, project_id, computer_id, repository)
);

INSERT OR IGNORE INTO pairing_project_checkouts (user_id, project_id, computer_id, repository, harness_project_id)
SELECT user_id, project_id, computer_id, '', harness_project_id FROM pairing_project_computers WHERE harness_project_id != '';

DELETE FROM pairing_project_computers WHERE position IS NULL;
ALTER TABLE pairing_project_computers DROP COLUMN harness_project_id;

-- Which of a project's repositories a play's runs work on, by its role: app or tests.
ALTER TABLE plays ADD COLUMN repository TEXT NOT NULL DEFAULT 'app';

-- The repository a queued run works on, as owner/name: the pull request's for a run an automation queued from a git
-- event, '' for the play's own.
ALTER TABLE play_queue ADD COLUMN repository TEXT NOT NULL DEFAULT '';
