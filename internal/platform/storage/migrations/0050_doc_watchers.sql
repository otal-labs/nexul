-- A doc's watchers are the people its change notifications go to (ADR 0101); watching 0 keeps an opt-out, so a later edit by that person does not re-add them.
CREATE TABLE IF NOT EXISTS doc_watchers (
    doc_id     TEXT NOT NULL REFERENCES docs(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL,
    watching   INTEGER NOT NULL DEFAULT 1,
    source     TEXT NOT NULL CHECK (source IN ('auto', 'manual')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (doc_id, user_id)
);

-- Every existing doc is watched by its creator and by everyone recorded as authoring one of its versions; an author who is not a user (empty, or a system id) is skipped.
INSERT OR IGNORE INTO doc_watchers (doc_id, user_id, watching, source, created_at, updated_at)
SELECT d.id, d.created_by, 1, 'auto', d.created_at, d.created_at
FROM docs d
WHERE d.created_by IN (SELECT id FROM users);

INSERT OR IGNORE INTO doc_watchers (doc_id, user_id, watching, source, created_at, updated_at)
SELECT v.doc_id, v.author_id, 1, 'auto', MIN(v.created_at), MIN(v.created_at)
FROM doc_versions v
WHERE v.author_id IN (SELECT id FROM users)
GROUP BY v.doc_id, v.author_id;
