-- A person's computers in one order, each with its own runs at once and chat folder (ADRs 0165, 0166).
ALTER TABLE pairing_computers ADD COLUMN position INTEGER NOT NULL DEFAULT 0;
ALTER TABLE pairing_computers ADD COLUMN runs_at_once INTEGER NOT NULL DEFAULT 1;
ALTER TABLE pairing_computers ADD COLUMN chat_project_id TEXT NOT NULL DEFAULT '';

-- The old default computer first, then the rest newest first.
UPDATE pairing_computers SET position = (
    SELECT COUNT(*) FROM pairing_computers o
    LEFT JOIN pairing_user_defaults d ON d.user_id = o.user_id
    WHERE o.user_id = pairing_computers.user_id AND o.id != pairing_computers.id AND (
        o.id IS d.default_computer_id
        OR (pairing_computers.id IS NOT d.default_computer_id AND (o.created_at > pairing_computers.created_at
            OR (o.created_at = pairing_computers.created_at AND o.id < pairing_computers.id)))
    )
);

-- Nobody's room shrinks: each computer takes the most runs at once any of its owner's workspaces allowed.
UPDATE pairing_computers SET runs_at_once = COALESCE((
    SELECT MAX(w.auto_play_runs_at_once) FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
    WHERE m.user_id = pairing_computers.user_id
), 1);

UPDATE pairing_computers SET chat_project_id = (
    SELECT d.fallback_project_id FROM pairing_user_defaults d WHERE d.default_computer_id = pairing_computers.id
) WHERE id IN (SELECT default_computer_id FROM pairing_user_defaults WHERE default_computer_id IS NOT NULL);

-- One row per computer a person named for a project: a position puts it in the project's own order, and a T3 project
-- is the checkout they picked there. Read by person and project for every run.
CREATE TABLE pairing_project_computers (
    user_id            TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id         TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    computer_id        TEXT NOT NULL REFERENCES pairing_computers(id) ON DELETE CASCADE,
    position           INTEGER,
    harness_project_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, project_id, computer_id)
);
INSERT INTO pairing_project_computers (user_id, project_id, computer_id, harness_project_id)
SELECT user_id, project_id, computer_id, harness_project_id FROM pairing_project_links;

CREATE TABLE pairing_project_links_new (
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    model_options TEXT NOT NULL DEFAULT '[]',
    start_in      TEXT NOT NULL DEFAULT '',
    placement     TEXT NOT NULL DEFAULT '',
    updated_at    INTEGER NOT NULL,
    PRIMARY KEY (user_id, project_id)
);
INSERT INTO pairing_project_links_new (user_id, project_id, provider, model, model_options, start_in, updated_at)
SELECT user_id, project_id, provider, model, model_options, start_in, updated_at FROM pairing_project_links;
DROP TABLE pairing_project_links;
ALTER TABLE pairing_project_links_new RENAME TO pairing_project_links;

CREATE TABLE pairing_user_defaults_new (
    user_id             TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    provider            TEXT NOT NULL DEFAULT '',
    model               TEXT NOT NULL DEFAULT '',
    model_options       TEXT NOT NULL DEFAULT '[]',
    start_in            TEXT NOT NULL DEFAULT '',
    placement           TEXT NOT NULL DEFAULT 'in_order',
    clone_automatically INTEGER NOT NULL DEFAULT 0
);
INSERT INTO pairing_user_defaults_new (user_id, provider, model, model_options, start_in)
SELECT user_id, provider, model, model_options, start_in FROM pairing_user_defaults;
DROP TABLE pairing_user_defaults;
ALTER TABLE pairing_user_defaults_new RENAME TO pairing_user_defaults;

ALTER TABLE workspaces DROP COLUMN auto_play_runs_at_once;

-- Why each computer could not take a waiting run, and on a trail the computers it passed over: JSON lists.
ALTER TABLE play_queue ADD COLUMN waiting_on TEXT NOT NULL DEFAULT '[]';
ALTER TABLE play_trails ADD COLUMN passed_over TEXT NOT NULL DEFAULT '[]';

-- Counts the play runs going on a computer, for its runs at once.
CREATE INDEX IF NOT EXISTS idx_play_trails_computer_active ON play_trails(computer_id, state);
