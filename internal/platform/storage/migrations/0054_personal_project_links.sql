-- A project's T3 link belongs to one person (ADR 0102); each existing link goes to its computer's owner, nobody else.
CREATE TABLE pairing_project_links_new (
    user_id            TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id         TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    computer_id        TEXT NOT NULL REFERENCES pairing_computers(id) ON DELETE CASCADE,
    harness_project_id TEXT NOT NULL,
    provider           TEXT NOT NULL DEFAULT '',
    model              TEXT NOT NULL DEFAULT '',
    model_options      TEXT NOT NULL DEFAULT '[]',
    updated_at         INTEGER NOT NULL,
    PRIMARY KEY (user_id, project_id)
);
INSERT INTO pairing_project_links_new (user_id, project_id, computer_id, harness_project_id, provider, model, model_options, updated_at)
SELECT c.user_id, l.project_id, l.computer_id, l.harness_project_id, l.provider, l.model, l.model_options, l.updated_at
FROM pairing_project_links l
JOIN pairing_computers c ON c.id = l.computer_id;
DROP TABLE pairing_project_links;
ALTER TABLE pairing_project_links_new RENAME TO pairing_project_links;
