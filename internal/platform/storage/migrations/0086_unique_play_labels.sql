-- An automation names a play by its label (ADR 0132), so a label is unique in its workspace, ignoring ASCII case as a
-- person reading the buttons would. Existing duplicates keep their creation order: the first keeps its label, the
-- next become "X (2)", "X (3)". The ranks are read before any rename, so a rename never shifts a later rank.
CREATE TABLE play_label_ranks (id TEXT PRIMARY KEY, n INTEGER NOT NULL);
INSERT INTO play_label_ranks (id, n)
SELECT p.id, 1 + (
    SELECT COUNT(*) FROM plays e
    WHERE e.workspace_id = p.workspace_id AND e.label = p.label COLLATE NOCASE
      AND (e.created_at < p.created_at OR (e.created_at = p.created_at AND e.id < p.id))
) FROM plays p;
UPDATE plays SET label = label || ' (' || (SELECT n FROM play_label_ranks r WHERE r.id = plays.id) || ')'
WHERE id IN (SELECT id FROM play_label_ranks WHERE n > 1);

-- A play already named "X (2)" beside two named "X" collides once more; the later of the two takes its id as well.
DELETE FROM play_label_ranks;
INSERT INTO play_label_ranks (id, n)
SELECT p.id, 1 + (
    SELECT COUNT(*) FROM plays e
    WHERE e.workspace_id = p.workspace_id AND e.label = p.label COLLATE NOCASE
      AND (e.created_at < p.created_at OR (e.created_at = p.created_at AND e.id < p.id))
) FROM plays p;
UPDATE plays SET label = label || ' (' || substr(id, 1, 8) || ')'
WHERE id IN (SELECT id FROM play_label_ranks WHERE n > 1);
DROP TABLE play_label_ranks;

-- Serves the label lookup an automation's runPlay makes, and refuses a second play of the same name.
CREATE UNIQUE INDEX IF NOT EXISTS idx_plays_label ON plays(workspace_id, label COLLATE NOCASE);

-- A run an automation queued through runPlay has no auto play; it names the automation instead.
ALTER TABLE play_queue ADD COLUMN automation_id TEXT NOT NULL DEFAULT '';

-- Makes a second match while one waits a no-op: at most one waiting row per auto play, or per automation and play,
-- per target.
DROP INDEX IF EXISTS idx_play_queue_waiting;
CREATE UNIQUE INDEX IF NOT EXISTS idx_play_queue_waiting ON play_queue(auto_play_id, automation_id, play_id, target_id)
    WHERE status IN ('queued', 'dispatching');

-- Running a play over the gateway now needs plays:run rather than plays:write; a token that ran plays keeps doing so.
UPDATE integration_installs SET scopes = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(integration_installs.scopes) UNION SELECT 'plays:run' ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(integration_installs.scopes) WHERE value = 'plays:write');

-- automations.scopes is JSON text in a BLOB, possibly empty; json_each would read a raw BLOB as JSONB.
UPDATE automations SET scopes = CAST((
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
        UNION SELECT 'plays:run' ORDER BY value
    )
) AS BLOB)
WHERE EXISTS (
    SELECT 1 FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
    WHERE value = 'plays:write'
);
