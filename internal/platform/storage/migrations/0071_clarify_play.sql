-- Workspaces made before the Clarify via AI play existed get it here, under its built-in key; new workspaces get it
-- from SeedDefaults.
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at)
SELECT lower(hex(randomblob(16))), w.id, 'Clarify via AI', 'doc', 'Asks the doc''s authors about the gaps in what they need, a round at a time, then writes the answers into the doc.', 'Clarify this doc for the people who wrote it. Read it with `doc_get`: its body and its clarification, every earlier round with its questions, answers, skipped questions, and "Anything else?" text. The instructions for this run, if any, win over everything below.

Find the gaps in what the doc says its authors need. Ask about what it must do: who uses it and what each of them does, the steps of each flow, the business rules and their exceptions, the data they have or must keep, what is in and out of scope, and what finished looks like. Ask about how well it must do it, in their terms: how many people use it and when, how fast it must feel, when it must be available, who may see what, devices and languages, accessibility, the systems it must work with, legal or industry rules, deadlines, budget, and what matters most. Never ask how to build it (hosting, databases, frameworks, architecture); those are the developer''s, so put anything technical left unclear in your reply.

Do not ask what an earlier round answered, and do not repeat a question still waiting for an answer. Ask a skipped question once more only if it still matters.

If gaps remain, post one round with `doc_update` `questions`: three to six, the biggest unknowns first, grouped by the doc''s sections. Write each in plain words for someone non-technical, with two to four concrete options, the one most projects pick first and labelled "(Suggested)", multi-select only where several can apply, and a one-line why that says why it matters to them. If the last round has "Anything else?" text, send `anything_else_reply` with it: one plain line that answers it, or says which of this round''s questions follow it up. Never use your question tool: post the round and end your turn.

If the developer could turn the doc into tickets without asking its authors anything more, there are no gaps left: send `doc_update` with `no_gaps` and the whole new body. Keep the authors'' headings and words, weave each answer into the section it belongs to, add a section only where nothing fits, and never mention questions or rounds. Questions skipped twice that still matter go under a short "Open points" section.

Never mention AI, an agent, or yourself in anything the authors see. Reply to the developer with what you asked and why, or that the doc is complete and what changed, plus any technical questions for them.', 1, NULL, '[]', 'clarify', '', strftime('%s','now'), strftime('%s','now')
FROM workspaces w
WHERE NOT EXISTS (SELECT 1 FROM plays p WHERE p.workspace_id = w.id AND p.builtin_key = 'clarify');
