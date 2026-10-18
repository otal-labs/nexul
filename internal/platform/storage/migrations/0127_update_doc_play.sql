-- Workspaces made before the Update doc via AI play existed get it here, under its built-in key; new workspaces get it
-- from SeedDefaults. Its label is the first of "Update doc via AI", " (2)", " (3)" no play holds in any case, since
-- play labels are unique per workspace without case.
WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 1000)
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at)
SELECT lower(hex(randomblob(16))), w.id,
    (SELECT iif(n.i = 1, 'Update doc via AI', 'Update doc via AI (' || n.i || ')') AS label FROM n
        WHERE NOT EXISTS (SELECT 1 FROM plays p WHERE p.workspace_id = w.id
            AND p.label = iif(n.i = 1, 'Update doc via AI', 'Update doc via AI (' || n.i || ')') COLLATE NOCASE)
        ORDER BY n.i LIMIT 1),
    'doc', 'Writes the answers to the doc''s questions into the doc, in its authors'' words, without asking more.', 'Write the answers to this doc''s questions into the doc, for the people who wrote it. Read it with `doc_get`: its body and its clarification, every round with its questions, answers, skipped questions, and "Anything else?" text. The instructions for this run, if any, win over everything below.

Ask nothing: never send `questions` and never use your question tool. If no question has an answer and no round has "Anything else?" text, change nothing and reply that there is nothing to write yet.

Otherwise send `doc_update` with `no_gaps` and the whole new body. Keep the authors'' headings and their own sentences wherever they still hold, weave each answer and each "Anything else?" text into the section it belongs to, add a section only where nothing fits, and never mention questions or rounds. The body may already hold earlier answers: change only what a newer or changed answer says. Add only what an answer says: never invent details, figures, examples, or requirements the authors did not give. Questions skipped or still unanswered that still matter go under a short "Open points" section. If the last round has "Anything else?" text with no reply, send `anything_else_reply` with it: one plain line that says where the doc now covers it; never name a round number in it.

Never mention AI, an agent, or yourself in anything the authors see. Reply to the developer with what changed in the doc and which points are still open, plus any technical questions for them.',
    1, NULL, '[]', 'update-doc', '', strftime('%s','now'), strftime('%s','now')
FROM workspaces w
WHERE NOT EXISTS (SELECT 1 FROM plays p WHERE p.workspace_id = w.id AND p.builtin_key = 'update-doc');
