-- A play's instruction text, one row per save that changed it; the newest 10 are kept. editor_ids is a JSON array of
-- user ids, via names the automation a save came through. builtin_default marks the seeded text nobody has edited.
CREATE TABLE IF NOT EXISTS play_instruction_versions (
    play_id         TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    version         INTEGER NOT NULL,
    body            TEXT NOT NULL,
    editor_ids      TEXT NOT NULL DEFAULT '[]',
    via             TEXT NOT NULL DEFAULT '',
    builtin_default INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL,
    PRIMARY KEY (play_id, version)
);

-- One row per changed field of a play other than its instructions; the newest 100 are kept. actor is a JSON object.
CREATE TABLE IF NOT EXISTS play_changes (
    id         TEXT PRIMARY KEY,
    play_id    TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    field      TEXT NOT NULL,
    from_value TEXT NOT NULL,
    to_value   TEXT NOT NULL,
    actor      TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

-- A play's history list, newest first, and the prune that keeps the newest rows.
CREATE INDEX IF NOT EXISTS idx_play_changes_play ON play_changes(play_id, created_at DESC);

-- The instructions version a run started with; null for runs before history began.
ALTER TABLE play_trails ADD COLUMN instructions_version INTEGER;

-- Every play starts with its current text as version 1. A built-in play no one has saved since it was created
-- (updated_at still equals created_at, which only a save moves) is the built-in default.
INSERT OR IGNORE INTO play_instruction_versions (play_id, version, body, editor_ids, via, builtin_default, created_at)
SELECT id, 1, instructions, '[]', '',
       CASE WHEN builtin_key != '' AND updated_at = created_at THEN 1 ELSE 0 END,
       CAST(strftime('%s', 'now') AS INTEGER)
FROM plays;
