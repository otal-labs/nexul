-- The flow's auto plays come seeded, switched off (ADR 0195). Fix with AI moves a backlog ticket to progress before it
-- builds, and To tickets assigns what it cuts to whoever ran it; a copy still on the text 0157 wrote gets the new one,
-- an edited copy keeps its own. A play this changes gets the new text as its newest version, in Go after this script.

UPDATE plays SET instructions = 'Build this ticket and carry it to the end.

1. Read it with `ticket_get`, which lists its blockers and its linked branches and pull requests, and with `message_list` its thread and the thread of any blocker whose body opens with "Needs a person". Stop and say why if this ticket''s own body opens with "Needs a person", or if a blocker is neither merged nor in a done column. If this ticket''s pull request merged and no failed test report came after, build nothing and go to step 6.
2. Work in the project checkout you are running in. If the ticket sits in a backlog column, move it to the first progress column with `ticket_update` before you start. Continue a linked branch of this repository whose pull request is absent or open; otherwise start a new branch `<ticket key>-<short-slug>` from the default branch. Fix what the latest failed test report names first.
3. Build what the acceptance criteria ask (for a bug, its expected result), and nothing else.
4. Prove it: run the project''s tests and linters, where it has any, then watch each criterion hold in the running app or with a command.
5. When every criterion holds, commit and push, open a pull request whose title starts with the ticket key unless one is already open, and link both to the ticket with `ticket_update`, passing `link_branch` and `link_pr`.
6. Conclude as the memories this prompt names for the end of the run say. With none: a merged ticket goes to the first testing column and keeps its branch only when it has a tester or a `test_target` url and the board has a testing column; any other merged ticket goes to the first done column and its branch is deleted. An open pull request stays open and its ticket goes to the first review column, or stays where it is when the board has none.

Reply with what changed, the proof for each criterion, and the pull request. If you could not finish, say what blocked you instead of opening a partial pull request.'
WHERE builtin_key = 'fix-with-ai' AND instructions = 'Build this ticket and carry it to the end.

1. Read it with `ticket_get`, which lists its blockers and its linked branches and pull requests, and with `message_list` its thread and the thread of any blocker whose body opens with "Needs a person". Stop and say why if this ticket''s own body opens with "Needs a person", or if a blocker is neither merged nor in a done column. If this ticket''s pull request merged and no failed test report came after, build nothing and go to step 6.
2. Work in the project checkout you are running in. Continue a linked branch of this repository whose pull request is absent or open; otherwise start a new branch `<ticket key>-<short-slug>` from the default branch. Fix what the latest failed test report names first.
3. Build what the acceptance criteria ask (for a bug, its expected result), and nothing else.
4. Prove it: run the project''s tests and linters, where it has any, then watch each criterion hold in the running app or with a command.
5. When every criterion holds, commit and push, open a pull request whose title starts with the ticket key unless one is already open, and link both to the ticket with `ticket_update`, passing `link_branch` and `link_pr`.
6. Conclude as the memories this prompt names for the end of the run say. With none: a merged ticket goes to the first testing column and keeps its branch only when it has a tester or a `test_target` url and the board has a testing column; any other merged ticket goes to the first done column and its branch is deleted. An open pull request stays open and its ticket goes to the first review column, or stays where it is when the board has none.

Reply with what changed, the proof for each criterion, and the pull request. If you could not finish, say what blocked you instead of opening a partial pull request.';

DELETE FROM instance_templates WHERE kind = 'play_instructions' AND key = 'fix-with-ai' AND body = 'Build this ticket and carry it to the end.

1. Read it with `ticket_get`, which lists its blockers and its linked branches and pull requests, and with `message_list` its thread and the thread of any blocker whose body opens with "Needs a person". Stop and say why if this ticket''s own body opens with "Needs a person", or if a blocker is neither merged nor in a done column. If this ticket''s pull request merged and no failed test report came after, build nothing and go to step 6.
2. Work in the project checkout you are running in. Continue a linked branch of this repository whose pull request is absent or open; otherwise start a new branch `<ticket key>-<short-slug>` from the default branch. Fix what the latest failed test report names first.
3. Build what the acceptance criteria ask (for a bug, its expected result), and nothing else.
4. Prove it: run the project''s tests and linters, where it has any, then watch each criterion hold in the running app or with a command.
5. When every criterion holds, commit and push, open a pull request whose title starts with the ticket key unless one is already open, and link both to the ticket with `ticket_update`, passing `link_branch` and `link_pr`.
6. Conclude as the memories this prompt names for the end of the run say. With none: a merged ticket goes to the first testing column and keeps its branch only when it has a tester or a `test_target` url and the board has a testing column; any other merged ticket goes to the first done column and its branch is deleted. An open pull request stays open and its ticket goes to the first review column, or stays where it is when the board has none.

Reply with what changed, the proof for each criterion, and the pull request. If you could not finish, say what blocked you instead of opening a partial pull request.';

UPDATE plays SET instructions = 'Turn this doc into tickets that can each be built, proven and shipped on their own.

1. Read the doc with `doc_get`, its thread with `message_list`, and the project with `project_get`: its columns, ticket types and their body templates. Find what is already tracked with `ticket_list`, once by `doc_id` and once with a `query` for the doc''s subject. On a re-run, create only what is missing.
2. Ask nothing, and leave technical choices to the builder. A product decision the doc leaves open, or a choice the project''s rules reserve for a person, becomes its own ticket: its body opens with "Needs a person: <the decision>. Settle it on this thread, then move the ticket to done.", and step 5 makes it the blocker of the tickets that wait on it.
3. Slice vertically: each ticket is one outcome someone can see, through every layer it needs, small enough for one pull request. The first is the thinnest end-to-end path.
4. Create them with `ticket_create`, the `doc_id` and `developer` `me`, each with the type that fits; `tickets` creates several of one type in one call. Fill the type''s body template: every acceptance criterion is a result someone can observe, and "How to verify" names the proof for each; for a bug, the expected result is the criterion and the steps to reproduce are the proof, and `ticket_create` takes `origin_id`, the ticket it was found in, or `origin_unknown` true.
5. Wire the order with `ticket_update` `add_blocker_ids`: a blocker only where a ticket cannot start before another lands. Leave them in the backlog.
6. Check coverage both ways: every requirement in the doc is in a ticket''s acceptance criteria or in your list of what you left out, and every ticket traces back to the doc.

Reply with the tickets in build order with their blockers, which can start now, the decisions left open, and what you left out and why.'
WHERE builtin_key = 'to-tickets-via-ai' AND instructions = 'Turn this doc into tickets that can each be built, proven and shipped on their own.

1. Read the doc with `doc_get`, its thread with `message_list`, and the project with `project_get`: its columns, ticket types and their body templates. Find what is already tracked with `ticket_list`, once by `doc_id` and once with a `query` for the doc''s subject. On a re-run, create only what is missing.
2. Ask nothing, and leave technical choices to the builder. A product decision the doc leaves open, or a choice the project''s rules reserve for a person, becomes its own ticket: its body opens with "Needs a person: <the decision>. Settle it on this thread, then move the ticket to done.", and step 5 makes it the blocker of the tickets that wait on it.
3. Slice vertically: each ticket is one outcome someone can see, through every layer it needs, small enough for one pull request. The first is the thinnest end-to-end path.
4. Create them with `ticket_create` and the `doc_id`, each with the type that fits; `tickets` creates several of one type in one call. Fill the type''s body template: every acceptance criterion is a result someone can observe, and "How to verify" names the proof for each; for a bug, the expected result is the criterion and the steps to reproduce are the proof, and `ticket_create` takes `origin_id`, the ticket it was found in, or `origin_unknown` true.
5. Wire the order with `ticket_update` `add_blocker_ids`: a blocker only where a ticket cannot start before another lands. Leave them in the backlog.
6. Check coverage both ways: every requirement in the doc is in a ticket''s acceptance criteria or in your list of what you left out, and every ticket traces back to the doc.

Reply with the tickets in build order with their blockers, which can start now, the decisions left open, and what you left out and why.';

DELETE FROM instance_templates WHERE kind = 'play_instructions' AND key = 'to-tickets-via-ai' AND body = 'Turn this doc into tickets that can each be built, proven and shipped on their own.

1. Read the doc with `doc_get`, its thread with `message_list`, and the project with `project_get`: its columns, ticket types and their body templates. Find what is already tracked with `ticket_list`, once by `doc_id` and once with a `query` for the doc''s subject. On a re-run, create only what is missing.
2. Ask nothing, and leave technical choices to the builder. A product decision the doc leaves open, or a choice the project''s rules reserve for a person, becomes its own ticket: its body opens with "Needs a person: <the decision>. Settle it on this thread, then move the ticket to done.", and step 5 makes it the blocker of the tickets that wait on it.
3. Slice vertically: each ticket is one outcome someone can see, through every layer it needs, small enough for one pull request. The first is the thinnest end-to-end path.
4. Create them with `ticket_create` and the `doc_id`, each with the type that fits; `tickets` creates several of one type in one call. Fill the type''s body template: every acceptance criterion is a result someone can observe, and "How to verify" names the proof for each; for a bug, the expected result is the criterion and the steps to reproduce are the proof, and `ticket_create` takes `origin_id`, the ticket it was found in, or `origin_unknown` true.
5. Wire the order with `ticket_update` `add_blocker_ids`: a blocker only where a ticket cannot start before another lands. Leave them in the backlog.
6. Check coverage both ways: every requirement in the doc is in a ticket''s acceptance criteria or in your list of what you left out, and every ticket traces back to the doc.

Reply with the tickets in build order with their blockers, which can start now, the decisions left open, and what you left out and why.';

-- Each workspace's Fix with AI and Test with AI gain the flow's auto plays, off, as SeedDefaults writes them, unless the
-- play already has one on the same moment and stage.
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, moment_stage, schedule, conditions, priority, once_within_minutes, run_on, computer_id, computer_ids, run_as, memory_ids, one_at_a_time, resource, created_by, created_at, updated_at)
SELECT lower(hex(randomblob(16))), p.id, p.workspace_id, 0, 'ticket.entered_stage', 'progress', NULL,
    '{"match":"all","groups":[]}', '{"rules":[],"otherwise":"normal"}', 0, 'developer', '', '[]', '', '[]', 0, '', '',
    CAST(strftime('%s', 'now') AS INTEGER), CAST(strftime('%s', 'now') AS INTEGER)
FROM plays p
WHERE p.builtin_key = 'fix-with-ai' AND NOT EXISTS (
    SELECT 1 FROM auto_plays a WHERE a.play_id = p.id AND a.moment = 'ticket.entered_stage' AND a.moment_stage = 'progress');

INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, moment_stage, schedule, conditions, priority, once_within_minutes, run_on, computer_id, computer_ids, run_as, memory_ids, one_at_a_time, resource, created_by, created_at, updated_at)
SELECT lower(hex(randomblob(16))), p.id, p.workspace_id, 0, 'ticket.unblocked', NULL, NULL,
    '{"match":"all","groups":[{"match":"all","rules":[{"field":"stage","op":"is","values":["backlog"]}]}]}',
    '{"rules":[],"otherwise":"normal"}', 0, 'developer', '', '[]', '', '[]', 1, '', '',
    CAST(strftime('%s', 'now') AS INTEGER), CAST(strftime('%s', 'now') AS INTEGER)
FROM plays p
WHERE p.builtin_key = 'fix-with-ai' AND NOT EXISTS (
    SELECT 1 FROM auto_plays a WHERE a.play_id = p.id AND a.moment = 'ticket.unblocked');

INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, moment_stage, schedule, conditions, priority, once_within_minutes, run_on, computer_id, computer_ids, run_as, memory_ids, one_at_a_time, resource, created_by, created_at, updated_at)
SELECT lower(hex(randomblob(16))), p.id, p.workspace_id, 0, 'ticket.entered_stage', 'testing', NULL,
    '{"match":"all","groups":[{"match":"all","rules":[{"field":"tester","op":"unset"}]}]}',
    '{"rules":[],"otherwise":"normal"}', 0, 'causer', '', '[]', '', '[]', 0, '', '',
    CAST(strftime('%s', 'now') AS INTEGER), CAST(strftime('%s', 'now') AS INTEGER)
FROM plays p
WHERE p.builtin_key = 'test-with-ai' AND NOT EXISTS (
    SELECT 1 FROM auto_plays a WHERE a.play_id = p.id AND a.moment = 'ticket.entered_stage' AND a.moment_stage = 'testing');
