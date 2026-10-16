-- A stack play's auto play runs as the person who picked the computer it lands on (ADR 0156): that computer and person,
-- and for a stack moment the service and outage that started the queued run.
ALTER TABLE auto_plays ADD COLUMN computer_id TEXT NOT NULL DEFAULT '';
ALTER TABLE auto_plays ADD COLUMN run_as TEXT NOT NULL DEFAULT '';
ALTER TABLE play_queue ADD COLUMN service TEXT NOT NULL DEFAULT '';
ALTER TABLE play_queue ADD COLUMN outage_id TEXT NOT NULL DEFAULT '';

-- A stack's own conversation, where its plays' runs report; one per stack, in the stack's project. An interview thread
-- stays one per project, so its index now names its kind.
ALTER TABLE conversations ADD COLUMN stack_id TEXT REFERENCES stacks(id) ON DELETE CASCADE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_stack ON conversations(stack_id) WHERE stack_id IS NOT NULL;
DROP INDEX IF EXISTS idx_conversations_interview;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_interview ON conversations(project_id) WHERE project_id IS NOT NULL AND kind = 'interview_thread';

-- Workspaces from before the stack play type get "Investigate a crash" here, under its built-in key; new workspaces get
-- it from SeedDefaults. Its label is the first of "Investigate a crash", " (2)", " (3)" no play holds in any case, since
-- play labels are unique per workspace without case.
WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 1000)
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at)
SELECT lower(hex(randomblob(16))), w.id,
    (SELECT iif(n.i = 1, 'Investigate a crash', 'Investigate a crash (' || n.i || ')') AS label FROM n
        WHERE NOT EXISTS (SELECT 1 FROM plays p WHERE p.workspace_id = w.id
            AND p.label = iif(n.i = 1, 'Investigate a crash', 'Investigate a crash (' || n.i || ')') COLLATE NOCASE)
        ORDER BY n.i LIMIT 1),
    'stack', 'Reads a crashed service''s logs and recent deploys, finds the cause, and writes it down. It never deploys.', 'A service in a stack crashed. Find out why and write it down. The stack is named below, and when a crash started this run so are the service and its outage. Read the stack with `stack_get`: its services, which of them are down now with their exit code and restart count, and its recent deploys. Pass `logs` with a service''s name to read that container''s last lines; the cause is usually in the output just before the exit. When no service is named below, read the logs of each service that is down. Then look at the recent deploys for what changed shortly before the crash, and find the cause in the project checkout you are running in: the commit, the config or the dependency that explains the exit code (137 with `oom` set is the container running out of memory). Write your findings where the team will see them. Look for an open ticket in the stack''s project that already names this crash, with `ticket_list` and a `query` of the service name, and add your findings to its thread with `message_post`; when there is none, create one with `ticket_create`, titled with the service and what happened, with the findings in its body. State the cause, or your best-supported guess when you cannot be sure, the evidence for it (log lines, the deploy, the commit), and what you would change. Say plainly when you could not find the cause. Never deploy, restart, stop or edit the stack or any of its services: this play only reads and reports. Open a fix pull request only if the instructions for this run say so. If the evidence points at Nexul itself, such as the runner, the deploy engine or how the stack was built, rather than at the application, say so in your findings and tell the person to triage it with the `nexul-triage` skill. Reply with the cause in one sentence and where you wrote it.',
    1, NULL, '[]', 'investigate-crash', '', strftime('%s','now'), strftime('%s','now')
FROM workspaces w
WHERE NOT EXISTS (SELECT 1 FROM plays p WHERE p.workspace_id = w.id AND p.builtin_key = 'investigate-crash');

-- Its auto play waits for a service to crash and is off until someone picks the computer it runs on and switches it on.
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, moment_stage, run_on, created_by, created_at, updated_at)
SELECT lower(hex(randomblob(16))), p.id, p.workspace_id, 0, 'stack.service_crashed', NULL, 'computer', '',
    strftime('%s','now'), strftime('%s','now')
FROM plays p
WHERE p.builtin_key = 'investigate-crash' AND NOT EXISTS (SELECT 1 FROM auto_plays a WHERE a.play_id = p.id);
