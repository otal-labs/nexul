-- The decisions check becomes an ordinary play in every workspace under its built-in key (ADR 0132); new workspaces get
-- it from SeedDefaults. Its label is the first of "Decisions check", " (2)", " (3)" no play holds in any case, since
-- play labels are unique per workspace without case.
WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 1000)
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at)
SELECT lower(hex(randomblob(16))), w.id,
    (SELECT iif(n.i = 1, 'Decisions check', 'Decisions check (' || n.i || ')') AS label FROM n
        WHERE NOT EXISTS (SELECT 1 FROM plays p WHERE p.workspace_id = w.id
            AND p.label = iif(n.i = 1, 'Decisions check', 'Decisions check (' || n.i || ')') COLLATE NOCASE)
        ORDER BY n.i LIMIT 1),
    'ticket', 'When a ticket reaches done, an agent decides whether it changed how the project works and logs it.', 'This ticket just reached done. Decide whether it changed how the project works: a new pattern, a library added or dropped, a rule or an earlier decision reversed. Routine work that follows the existing patterns changes nothing. Read the ticket and its linked pull requests with `ticket_get`, each pull request with `pull_request_get`, and the project''s decisions log, the memory of kind `decisions_log` that `memory_list` finds, with `memory_get`. If nothing changed, reply "No decision recorded" with one line on why, and write nothing. Otherwise add one entry of at most three lines as its own paragraph at the end of the log: `<YYYY-MM-DD> — <the decision in one line>`, then `Why: <one line>`, then `Ticket: [<ticket key>](/tickets/<ticket id>)`. When the decision reverses an earlier entry, append ` (superseded by <ticket key>)` to that entry''s first line instead of deleting it, and leave every other entry as it is. Save with `memory_update` on the log; if the project has no log yet, create it with `memory_create` passing `kind` `decisions_log` and the project id. Reply with the entry you wrote.',
    1, 'done', '[]', 'decisions-check', '', strftime('%s','now'), strftime('%s','now')
FROM workspaces w
WHERE NOT EXISTS (SELECT 1 FROM plays p WHERE p.workspace_id = w.id AND p.builtin_key = 'decisions-check');

-- Its auto play runs it on whoever moves a ticket into done, switched on where the old per-workspace switch was on;
-- workspaces.decisions_check_enabled is read here for the last time and stays in place unused.
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, moment_stage, run_on, created_by, created_at, updated_at)
SELECT lower(hex(randomblob(16))), p.id, p.workspace_id, w.decisions_check_enabled, 'ticket.entered_stage', 'done', 'causer', '',
    strftime('%s','now'), strftime('%s','now')
FROM plays p JOIN workspaces w ON w.id = p.workspace_id
WHERE p.builtin_key = 'decisions-check' AND NOT EXISTS (SELECT 1 FROM auto_plays a WHERE a.play_id = p.id);

-- The check's past trails name its play, so the "didn't run" retry and a continued run find it.
UPDATE play_trails SET play_id = (
    SELECT p.id FROM plays p WHERE p.workspace_id = play_trails.workspace_id AND p.builtin_key = 'decisions-check'
)
WHERE play_id = 'decisions-check';
