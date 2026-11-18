-- The starting templates carry one flow from interview to merge (ADR 0193). A copy still holding the default it was
-- seeded with gets the new one; an edited copy keeps its own, and an instance template saved with the previous default
-- follows the code default again. A play this changes gets the new text as its newest version, in Go after this script.

UPDATE plays SET instructions = 'Build this ticket and carry it to the end.

1. Read it with `ticket_get`, which lists its blockers and its linked branches and pull requests, and with `message_list` its thread and the thread of any blocker whose body opens with "Needs a person". Stop and say why if this ticket''s own body opens with "Needs a person", or if a blocker is neither merged nor in a done column. If this ticket''s pull request merged and no failed test report came after, build nothing and go to step 6.
2. Work in the project checkout you are running in. Continue a linked branch of this repository whose pull request is absent or open; otherwise start a new branch `<ticket key>-<short-slug>` from the default branch. Fix what the latest failed test report names first.
3. Build what the acceptance criteria ask (for a bug, its expected result), and nothing else.
4. Prove it: run the project''s tests and linters, where it has any, then watch each criterion hold in the running app or with a command.
5. When every criterion holds, commit and push, open a pull request whose title starts with the ticket key unless one is already open, and link both to the ticket with `ticket_update`, passing `link_branch` and `link_pr`.
6. Conclude as the memories this prompt names for the end of the run say. With none: a merged ticket goes to the first testing column and keeps its branch only when it has a tester or a `test_target` url and the board has a testing column; any other merged ticket goes to the first done column and its branch is deleted. An open pull request stays open and its ticket goes to the first review column, or stays where it is when the board has none.

Reply with what changed, the proof for each criterion, and the pull request. If you could not finish, say what blocked you instead of opening a partial pull request.'
WHERE builtin_key = 'fix-with-ai' AND instructions = 'Confirm your Nexul access by reading the ticket with `ticket_get`, which also lists its linked pull requests and branches. Work in the project checkout you are running in. Create a branch named `<ticket key>-<short-slug>` from the default branch, implement the fix, run the project''s tests and linters, and commit. Push the branch, open a pull request whose title starts with the ticket key, then link both to the ticket with `ticket_update`, passing `link_branch` and `link_pr`. Do not merge unless a memory selected for this run explicitly permits merging; if one does, merge once checks pass. Reply with what changed, how it was verified, and the PR link. If you cannot complete the fix, say what blocked you instead of opening a partial PR.';

DELETE FROM instance_templates WHERE kind = 'play_instructions' AND key = 'fix-with-ai' AND body = 'Confirm your Nexul access by reading the ticket with `ticket_get`, which also lists its linked pull requests and branches. Work in the project checkout you are running in. Create a branch named `<ticket key>-<short-slug>` from the default branch, implement the fix, run the project''s tests and linters, and commit. Push the branch, open a pull request whose title starts with the ticket key, then link both to the ticket with `ticket_update`, passing `link_branch` and `link_pr`. Do not merge unless a memory selected for this run explicitly permits merging; if one does, merge once checks pass. Reply with what changed, how it was verified, and the PR link. If you cannot complete the fix, say what blocked you instead of opening a partial PR.';

UPDATE plays SET instructions = 'Turn this doc into tickets that can each be built, proven and shipped on their own.

1. Read the doc with `doc_get`, its thread with `message_list`, and the project with `project_get`: its columns, ticket types and their body templates. Find what is already tracked with `ticket_list`, once by `doc_id` and once with a `query` for the doc''s subject. On a re-run, create only what is missing.
2. Ask nothing, and leave technical choices to the builder. A product decision the doc leaves open, or a choice the project''s rules reserve for a person, becomes its own ticket: its body opens with "Needs a person: <the decision>. Settle it on this thread, then move the ticket to done.", and step 5 makes it the blocker of the tickets that wait on it.
3. Slice vertically: each ticket is one outcome someone can see, through every layer it needs, small enough for one pull request. The first is the thinnest end-to-end path.
4. Create them with `ticket_create` and the `doc_id`, each with the type that fits; `tickets` creates several of one type in one call. Fill the type''s body template: every acceptance criterion is a result someone can observe, and "How to verify" names the proof for each; for a bug, the expected result is the criterion and the steps to reproduce are the proof, and `ticket_create` takes `origin_id`, the ticket it was found in, or `origin_unknown` true.
5. Wire the order with `ticket_update` `add_blocker_ids`: a blocker only where a ticket cannot start before another lands. Leave them in the backlog.
6. Check coverage both ways: every requirement in the doc is in a ticket''s acceptance criteria or in your list of what you left out, and every ticket traces back to the doc.

Reply with the tickets in build order with their blockers, which can start now, the decisions left open, and what you left out and why.'
WHERE builtin_key = 'to-tickets-via-ai' AND instructions = 'Confirm your Nexul access by reading the doc with `doc_get`. Read the doc''s project with `project_get`, which lists its status columns and ticket types. Search existing tickets with `ticket_list` and a `query` so you do not duplicate work already tracked. Split the doc into tickets a developer could pick up independently: one outcome per ticket, a title under eighty characters, a body with context, acceptance criteria, and a pointer to the doc section it came from. Create each with `ticket_create`, passing the doc id so the ticket links back. Put them in the backlog column. Reply with the list of tickets created and anything in the doc you deliberately did not turn into a ticket.';

DELETE FROM instance_templates WHERE kind = 'play_instructions' AND key = 'to-tickets-via-ai' AND body = 'Confirm your Nexul access by reading the doc with `doc_get`. Read the doc''s project with `project_get`, which lists its status columns and ticket types. Search existing tickets with `ticket_list` and a `query` so you do not duplicate work already tracked. Split the doc into tickets a developer could pick up independently: one outcome per ticket, a title under eighty characters, a body with context, acceptance criteria, and a pointer to the doc section it came from. Create each with `ticket_create`, passing the doc id so the ticket links back. Put them in the backlog column. Reply with the list of tickets created and anything in the doc you deliberately did not turn into a ticket.';

UPDATE plays SET instructions = 'Finish this project''s interview: the person answered the Interview questions on the project''s Interview page, and you settle what is still open, then write the project''s rules. The project, the answers it already records, and the checkouts of its project sources under question are named below.

1. Call `memory_create` with the project id and `kind` `interview`, and nothing else. It returns the interview memory: its body is the current rules (empty the first time), with its `questions`, its `answers` (round 0 answers the questions, later rounds are earlier follow-ups, and an answer to a question no longer listed still counts) and its `sources`.
2. Find the facts yourself. Read the checkout you are running in: manifests, CI, linter and formatter config, tests, README and decision records. Read every source whose stance is question: a path in the checkout, a doc with `doc_get`, a memory with `memory_get`, pasted text, or another project through `memory_list` and `memory_get`, plus its checkout when one is named below. Read a large source by its headings first. A source whose stance is follow is context only; write no drafts.
3. Ask only for decisions: a question the person skipped whose answer neither the code nor its hint settles, a gap that changes how work is done here, an answer the code contradicts, and what a question source did that the answers leave open. Never ask what you can read or what the project already records. An answer stands: do not ask it again in other words.
4. Ask with your own question tool, never through another agent, a round at a time: all of a round''s questions in one call, four at most, each with a header of at most 12 characters naming its topic, your recommended answer first and labelled (Recommended), and its question text followed by one sentence on why you ask, naming what you found and where. Stop after two rounds in this run, and sooner when the code or the recorded answers settle the rest.
5. Write the memory with `memory_update`, passing its id and the full markdown body: rules, not a transcript, as short imperative lines under headings you choose. Record a fact only where an agent would get it wrong without it, and a rule only when the person gave or confirmed it, except what step 6 takes from the code. Keep every rule no answer contradicts.
6. Always record how a change reaches the main branch (the answer, or the code when that question was skipped), which tests a change needs (those ticked, with any coverage floor; none when skipped), and what an agent may do without asking, which is only what the person ticked: skipped, it asks first, merging its own pull request included.
7. The body is capped at 8,000 characters and every agent turn reads it first, so leave out what a linter or the code already enforces.

Reply with what the memory now says and what this run changed.'
WHERE builtin_key = 'interview' AND instructions = 'Run this project''s interview follow-ups: the person has answered the workspace''s Interview template on the project''s Interview page, and you ask about what is still open, then write the interview memory; the project, the answers it already records, and the checkouts of its project sources under question are named below. Start by calling `memory_create` with the project id and `kind` `interview`: it returns the interview memory, whose body is the project''s current rules (empty the first time), its `questions` (the Interview template''s questions), its `answers` (the stored answers: round 0 answers the template''s questions, round 1 and up are earlier follow-ups), and its `sources`. An answer to a question no longer in `questions` answers an earlier wording of one; use it too. Then read the checkout you are running in: manifests and lockfiles, CI config, linter and formatter config, tests, README, docs and decision records. Read every source whose stance is question as well: how something was done before, such as an earlier phase of this project. A source is a path in the checkout you are running in, a doc with `doc_get`, a memory with `memory_get`, pasted text from its body, or another project through its memories and interview answers with `memory_list` and `memory_get`, plus its checkout when one is named below. Read a large source selectively: its table of contents or headings first, then only the passages a question needs. Read a source whose stance is follow only as context for the gaps; another run drafts answers from it, so write no drafts. Ask about every skipped or unanswered question, every gap the answers leave that the code cannot settle, anything the code contradicts, and what a question source did that the answers do not settle. Ask with your own question tool, never through another agent, one round at a time: all of a round''s questions in one call, each with your recommended answer as its first option, labelled (Recommended), and a header of at most 12 characters naming its topic, such as Testing. Write each question''s text as the question followed by one sentence on why you are asking that names what you found and where, such as: When are tests written? You skipped this, and most commits in the last month add a test file beside the code they change. For a question source, name the source and the file, such as: Keep booking state on the server for phase 2? Phase 1 keeps it in the mobile app, in src/state/booking.ts of its checkout. Keep asking rounds until nothing is left open. Never record something from the code or a source the person did not confirm, and never ask whether to scan the codebase. Do not ask again what the project already records, such as where its tests live, unless the person changes it; record a change there with `project_update` and `tests_location` as well. Then write the memory with `memory_update`, passing its id and the full markdown body: rules, not a transcript, as short imperative lines under headings you choose, with no questions, answers, or narration. Keep every existing rule no answer contradicts, and change only what the answers change. The body is capped at 8,000 characters and every agent turn in the project reads it first, so keep it well under the cap: tighten wording and drop what a linter or the code already enforces. Reply with a short summary of what the memory now says and what this run changed.';

DELETE FROM instance_templates WHERE kind = 'play_instructions' AND key = 'interview' AND body = 'Run this project''s interview follow-ups: the person has answered the workspace''s Interview template on the project''s Interview page, and you ask about what is still open, then write the interview memory; the project, the answers it already records, and the checkouts of its project sources under question are named below. Start by calling `memory_create` with the project id and `kind` `interview`: it returns the interview memory, whose body is the project''s current rules (empty the first time), its `questions` (the Interview template''s questions), its `answers` (the stored answers: round 0 answers the template''s questions, round 1 and up are earlier follow-ups), and its `sources`. An answer to a question no longer in `questions` answers an earlier wording of one; use it too. Then read the checkout you are running in: manifests and lockfiles, CI config, linter and formatter config, tests, README, docs and decision records. Read every source whose stance is question as well: how something was done before, such as an earlier phase of this project. A source is a path in the checkout you are running in, a doc with `doc_get`, a memory with `memory_get`, pasted text from its body, or another project through its memories and interview answers with `memory_list` and `memory_get`, plus its checkout when one is named below. Read a large source selectively: its table of contents or headings first, then only the passages a question needs. Read a source whose stance is follow only as context for the gaps; another run drafts answers from it, so write no drafts. Ask about every skipped or unanswered question, every gap the answers leave that the code cannot settle, anything the code contradicts, and what a question source did that the answers do not settle. Ask with your own question tool, never through another agent, one round at a time: all of a round''s questions in one call, each with your recommended answer as its first option, labelled (Recommended), and a header of at most 12 characters naming its topic, such as Testing. Write each question''s text as the question followed by one sentence on why you are asking that names what you found and where, such as: When are tests written? You skipped this, and most commits in the last month add a test file beside the code they change. For a question source, name the source and the file, such as: Keep booking state on the server for phase 2? Phase 1 keeps it in the mobile app, in src/state/booking.ts of its checkout. Keep asking rounds until nothing is left open. Never record something from the code or a source the person did not confirm, and never ask whether to scan the codebase. Do not ask again what the project already records, such as where its tests live, unless the person changes it; record a change there with `project_update` and `tests_location` as well. Then write the memory with `memory_update`, passing its id and the full markdown body: rules, not a transcript, as short imperative lines under headings you choose, with no questions, answers, or narration. Keep every existing rule no answer contradicts, and change only what the answers change. The body is capped at 8,000 characters and every agent turn in the project reads it first, so keep it well under the cap: tighten wording and drop what a linter or the code already enforces. Reply with a short summary of what the memory now says and what this run changed.';

UPDATE plays SET instructions = 'Draft answers to this project''s Interview questions from its follow sources, for the person to confirm on the project''s Interview page. The project, this run''s trail id, and the checkouts of its project sources are named below.

1. Call `memory_create` with the project id and `kind` `interview`, and nothing else. It returns the interview memory with its `questions`, its `answers` (round 0 answers the questions; a skip is no answer), its `sources` and its `drafts`.
2. Read every source whose stance is follow: a path in the checkout you are running in, a doc with `doc_get`, a memory with `memory_get`, pasted text, or another project through `memory_list` and `memory_get`, plus its checkout when one is named below. Read a large source by its headings first. Never read or draft from a source whose stance is question.
3. Draft each unanswered or skipped question a follow source speaks to. Draft an answered question only where the sources now disagree with its answer. No source, no draft.
4. Shape a draft like an answer: `question` with the question''s exact text, `selected` with the option labels it picks, `text` for what the options do not say, `source_ids`, and `where`, one line of at most 500 characters saying where inside the source, such as a heading or a quoted phrase, without repeating the source''s name.
5. Save drafts with `memory_update` on the interview memory, in `drafts` with this run''s `trail_id`, as you find them, so the page shows them while you work. A draft replaces its question''s earlier one.

Ask nothing and use no question tool. Change nothing else: not the memory''s body, its answers or its sources. End with one line saying how many questions you drafted and from which sources.'
WHERE builtin_key = 'interview-draft' AND instructions = 'Draft answers to this project''s Interview template from its follow sources, for the person to confirm on the project''s Interview page; the project, this run''s trail id, and the checkouts of its project sources are named below. Start by calling `memory_create` with the project id and `kind` `interview`: it returns the interview memory with its `questions` (the Interview template''s questions), its `answers` (round 0 answers the template''s questions, a skip is no answer), its `sources`, and its `drafts`. Read every source whose stance is follow: a path in the checkout you are running in, a doc with `doc_get`, a memory with `memory_get`, pasted text from its body, and another project through its memories and interview answers with `memory_list` and `memory_get`, plus its checkout when one is named below. Read a large source selectively: its table of contents or headings first, then only the passages a question needs. Never read a source whose stance is question and never draft from one; another run asks about those. Draft every template question with no answer or a skip that a follow source speaks to. Draft a question that already has an answer only where the sources now say something different from that answer, and leave every other answered question alone. A question no follow source speaks to gets no draft. Shape each draft like an answer: `selected` with the labels of the options it picks, `text` for what the options do not say, `source_ids` with the ids of the follow sources it came from, and `where`, one line of at most 500 characters saying where inside them: a heading, a file inside a folder source, or a quoted phrase, such as Testing, "Test error paths first". The page shows it after the sources'' names, so never repeat a source''s name in it. Save drafts with `memory_update` on the interview memory, in `drafts` with this run''s `trail_id`, as you find them rather than all at the end, so the page shows them while you work; a draft replaces its question''s earlier one. Never ask the person anything and never use a question tool: the person confirms drafts on the page. Do not change the memory''s body, its answers, or its sources. End with a one-line summary of how many questions you drafted and from which sources.';

DELETE FROM instance_templates WHERE kind = 'play_instructions' AND key = 'interview-draft' AND body = 'Draft answers to this project''s Interview template from its follow sources, for the person to confirm on the project''s Interview page; the project, this run''s trail id, and the checkouts of its project sources are named below. Start by calling `memory_create` with the project id and `kind` `interview`: it returns the interview memory with its `questions` (the Interview template''s questions), its `answers` (round 0 answers the template''s questions, a skip is no answer), its `sources`, and its `drafts`. Read every source whose stance is follow: a path in the checkout you are running in, a doc with `doc_get`, a memory with `memory_get`, pasted text from its body, and another project through its memories and interview answers with `memory_list` and `memory_get`, plus its checkout when one is named below. Read a large source selectively: its table of contents or headings first, then only the passages a question needs. Never read a source whose stance is question and never draft from one; another run asks about those. Draft every template question with no answer or a skip that a follow source speaks to. Draft a question that already has an answer only where the sources now say something different from that answer, and leave every other answered question alone. A question no follow source speaks to gets no draft. Shape each draft like an answer: `selected` with the labels of the options it picks, `text` for what the options do not say, `source_ids` with the ids of the follow sources it came from, and `where`, one line of at most 500 characters saying where inside them: a heading, a file inside a folder source, or a quoted phrase, such as Testing, "Test error paths first". The page shows it after the sources'' names, so never repeat a source''s name in it. Save drafts with `memory_update` on the interview memory, in `drafts` with this run''s `trail_id`, as you find them rather than all at the end, so the page shows them while you work; a draft replaces its question''s earlier one. Never ask the person anything and never use a question tool: the person confirms drafts on the page. Do not change the memory''s body, its answers, or its sources. End with a one-line summary of how many questions you drafted and from which sources.';

UPDATE plays SET instructions = 'Audit code against this project''s interview memory and write the findings as one doc. The project, today''s date, this run''s trail id, and the code to audit are named below.

1. Call `memory_create` with the project id and `kind` `interview`, and nothing else. Its body is the project''s rules, the yardstick, and it returns the `answers` and `sources` too. With an empty body, stop and say the interview is not written yet.
2. Read the code named below: each project source under question through its checkout on this computer, or through its memories and interview answers with `memory_list` and `memory_get` when no checkout is named. With no source under question, audit this project''s own code in the checkout you are running in. Read first what the answers say each source under question did; then structure, manifests and config; then the code each rule speaks to.
3. Write one new doc with `doc_create` in this project, with no `folder_id`, titled Audit of <what you audited>, <today''s date>. Open it with one line naming what you audited and this run''s trail id. Group the findings under the memory''s headings, in the memory''s order, one line each: the verdict, where it is, as `path:line` when it has one, and the rule it meets or breaks. A heading with nothing found says so in one line.
4. Auditing a source under question, the verdicts are Carry over (it meets the rule), Rebuild (this project needs it, but it breaks the rule) and Avoid (the rules rule it out). Auditing this project''s own code, they are Keeps (it meets the rule) and Fix (it breaks it).

Ask nothing, file no tickets, and change no earlier audit, the memory, its answers or its sources. End with one line naming the doc by its title and counting its findings per verdict.'
WHERE builtin_key = 'audit' AND instructions = 'Audit code against this project''s interview memory and write what you find as one doc; the project, today''s date, this run''s trail id, and the code to audit are named below. Start by calling `memory_create` with the project id and `kind` `interview`: it returns the interview memory, whose body is the project''s rules under its headings, with its `answers` and its `sources`. The memory is the yardstick. Check the known breaks first: the answers to follow-ups about what a source under question did, such as one naming how the old project keeps booking state and asking what this one does instead. Then read the code named below: each project source under question through its checkout on this computer, or through its memories and interview answers with `memory_list` and `memory_get` when no checkout is named; with no project source under question, audit this project''s own code in the checkout you are running in. Read selectively: structure, manifests, and config first, then the code each rule speaks to. Write one doc with `doc_create` in this project, without a `folder_id` so it lands in the Main folder, titled Audit of <what you audited>, <today''s date>, such as Audit of Booking app, 2026-10-04. Open its body with one line naming what you audited and this run''s trail id. Group the findings under the memory''s headings, in the memory''s order, one line per finding: its verdict, where it is as `path:line`, and the rule it meets or breaks. A heading with nothing found says so in one line. Auditing a source under question, the verdicts are Carry over (it meets the rule, bring it across), Rebuild (it does a job this project needs but breaks the rule), and Avoid (the rules rule it out). Auditing this project''s own code, they are Keeps (it meets the rule) and Fix (it breaks it). Write a new doc every run, and change no earlier audit, the memory, its answers, or its sources. File no tickets: a person turns the doc into work with To tickets via AI. Never ask the person anything and never use a question tool. End with a one-line summary naming the doc by its title and counting its findings per verdict.';

DELETE FROM instance_templates WHERE kind = 'play_instructions' AND key = 'audit' AND body = 'Audit code against this project''s interview memory and write what you find as one doc; the project, today''s date, this run''s trail id, and the code to audit are named below. Start by calling `memory_create` with the project id and `kind` `interview`: it returns the interview memory, whose body is the project''s rules under its headings, with its `answers` and its `sources`. The memory is the yardstick. Check the known breaks first: the answers to follow-ups about what a source under question did, such as one naming how the old project keeps booking state and asking what this one does instead. Then read the code named below: each project source under question through its checkout on this computer, or through its memories and interview answers with `memory_list` and `memory_get` when no checkout is named; with no project source under question, audit this project''s own code in the checkout you are running in. Read selectively: structure, manifests, and config first, then the code each rule speaks to. Write one doc with `doc_create` in this project, without a `folder_id` so it lands in the Main folder, titled Audit of <what you audited>, <today''s date>, such as Audit of Booking app, 2026-10-04. Open its body with one line naming what you audited and this run''s trail id. Group the findings under the memory''s headings, in the memory''s order, one line per finding: its verdict, where it is as `path:line`, and the rule it meets or breaks. A heading with nothing found says so in one line. Auditing a source under question, the verdicts are Carry over (it meets the rule, bring it across), Rebuild (it does a job this project needs but breaks the rule), and Avoid (the rules rule it out). Auditing this project''s own code, they are Keeps (it meets the rule) and Fix (it breaks it). Write a new doc every run, and change no earlier audit, the memory, its answers, or its sources. File no tickets: a person turns the doc into work with To tickets via AI. Never ask the person anything and never use a question tool. End with a one-line summary naming the doc by its title and counting its findings per verdict.';

UPDATE plays SET instructions = 'Test this ticket the way a tester would, then pass or fail it.

1. Read it with `ticket_get` and its thread with `message_list`. Test its acceptance criteria (for a bug, its expected result). Never test on production. With a `test_target` url, test there. With none, test this checkout on the ticket''s linked branch while its pull request is open, else the default branch; if a criterion needs the running app, stop without a result and reply: No test environment yet. Testing never uses production, so add a branch deploy rule with its own network.
2. Check each criterion in turn, the way "How to verify" says where the ticket has it. Note what you did and what you saw.
3. Run the project''s tests, where it has any: in the repository with the role tests when `project_get` lists one, cloned if needed, on its default branch; else in this checkout, on the ticket''s linked branch while its pull request is open, else the default branch.
4. Only where the testing rules in the project''s interview memory call for an end-to-end suite and the ticket has a `test_target` url, add or extend a test for the criteria and run it there. Push it in the repository the tests live in: to the ticket''s linked branch there while its pull request is open, otherwise to a new branch `<ticket key>-tests` with a pull request whose title starts with the ticket key, linked to the ticket with `ticket_update`. Merge it only as the project''s rules allow, after a review by someone who did not write it.
5. Record the result with `ticket_test_report`: `outcome` `pass` when every criterion held and the tests you ran passed, otherwise `fail` with the steps to reproduce, the expected result and the actual one. A pass moves the ticket to done; a fail posts the report to its thread and moves it back to progress. After a pass, delete each linked branch whose pull request merged, which takes its preview down; for one still open, keep it and post a note on the ticket that it passed and still needs merging.

Reply with each criterion and whether it held, the tests you ran and added, and the result you recorded. If you could not reach the url or run the tests, say what blocked you and record nothing.'
WHERE builtin_key = 'test-with-ai' AND instructions = 'Test this ticket the way a tester would, then pass or fail it. Read it with `ticket_get`, which also carries its links and where to test; its acceptance criteria are what you test against. Follow the testing strategy in this project''s interview memory, which comes with this run. With no interview, check each criterion on the live URL and run the tests the project already has, and add none. Where to test is the `test_target` in that result. Test nowhere else: never production, and never anything that shares production''s services. If its url is empty, the only place to test is production: stop without passing or failing the ticket, and reply that it needs a deploy branch on its own network. Open the url and check it against each acceptance criterion in turn, noting what you did and what you saw. Run the project''s tests. List its repositories with `project_get`: when one has the role tests, run them from that repository, cloning it if the checkout you are running in is not it; otherwise run them in this checkout on the ticket''s linked branch. Where the interview calls for an automated end-to-end suite, add or extend a test covering the ticket''s acceptance criteria, run it against the url, commit it, and push: to the ticket''s linked branch, or in a tests repository to a branch named `<ticket key>-<short-slug>` with a pull request whose title starts with the ticket key. Then record the result exactly as a person would. If every criterion holds and the tests pass, call `ticket_test_report` with `outcome` `pass`. Otherwise call it with `outcome` `fail` and the bug template filled: the steps to reproduce, the expected result the criterion promises, and the actual result you saw. It posts them to the ticket''s thread and moves the ticket back to progress. Reply with each criterion and whether it held, the tests you ran and added, and the result you recorded. If you cannot reach the url or run the tests, say what blocked you instead of passing or failing the ticket.';

DELETE FROM instance_templates WHERE kind = 'play_instructions' AND key = 'test-with-ai' AND body = 'Test this ticket the way a tester would, then pass or fail it. Read it with `ticket_get`, which also carries its links and where to test; its acceptance criteria are what you test against. Follow the testing strategy in this project''s interview memory, which comes with this run. With no interview, check each criterion on the live URL and run the tests the project already has, and add none. Where to test is the `test_target` in that result. Test nowhere else: never production, and never anything that shares production''s services. If its url is empty, the only place to test is production: stop without passing or failing the ticket, and reply that it needs a deploy branch on its own network. Open the url and check it against each acceptance criterion in turn, noting what you did and what you saw. Run the project''s tests. List its repositories with `project_get`: when one has the role tests, run them from that repository, cloning it if the checkout you are running in is not it; otherwise run them in this checkout on the ticket''s linked branch. Where the interview calls for an automated end-to-end suite, add or extend a test covering the ticket''s acceptance criteria, run it against the url, commit it, and push: to the ticket''s linked branch, or in a tests repository to a branch named `<ticket key>-<short-slug>` with a pull request whose title starts with the ticket key. Then record the result exactly as a person would. If every criterion holds and the tests pass, call `ticket_test_report` with `outcome` `pass`. Otherwise call it with `outcome` `fail` and the bug template filled: the steps to reproduce, the expected result the criterion promises, and the actual result you saw. It posts them to the ticket''s thread and moves the ticket back to progress. Reply with each criterion and whether it held, the tests you ran and added, and the result you recorded. If you cannot reach the url or run the tests, say what blocked you instead of passing or failing the ticket.';

UPDATE plays SET instructions = 'Find what this doc leaves open and ask its authors. Read it with `doc_get`: its body, its clarification and every earlier round. The instructions for this run, if any, win over everything below.

A gap is something the developer who ran this would have to guess to build it: who uses it and what each of them does, the steps of a flow, a rule and its exceptions, the data kept, what is in and out of scope, what finished looks like, and how well it must work in the authors'' terms, such as how many people, how fast, who may see what, which devices, by when, and what matters most. Never ask how to build it; put anything technical left unclear in your reply to the developer.

If the previous round has "Anything else?" text and no reply yet, send `anything_else_reply` on the `doc_update` this run sends, or alone when it posts no questions: one plain line that answers it, names which new question follows it up, or says where the doc now covers it, with no round number.

If gaps remain, post one round with `doc_update` `questions`: one to six, the biggest unknown first, one gap per question. Write each in plain words for someone non-technical, with two to four options, the one most projects pick listed first and labelled "(Suggested)", multi-select only where options combine, and a one-line why that says why it matters to them. Never repeat a question that was answered or is still waiting; ask a skipped one once more only if it still matters; a gap left out of an earlier round for space still needs asking. Never use your question tool: post that update and end your turn.

If a question is still waiting and nothing else is open, post no questions and say so in your reply. If none is waiting and the developer could turn the doc into tickets without asking its authors anything more, ask nothing: send `doc_update` with `no_gaps` and the whole new body. Keep the authors'' headings and their own sentences wherever they still hold, put each answer and each "Anything else?" text in the section it belongs to, adding a section only where nothing fits, and add only what an answer says, with no invented detail, figure or requirement. Questions asked and skipped twice that still matter go under a short "Open points" section. Never mention questions or rounds in the doc.

Never mention AI, an agent, or yourself in anything the authors see. Reply to the developer with what you asked and why, or that the doc is complete and what changed, plus any technical questions for them.'
WHERE builtin_key = 'clarify' AND instructions = 'Clarify this doc for the people who wrote it. Read it with `doc_get`: its body and its clarification, every earlier round with its questions, answers, skipped questions, and "Anything else?" text. The instructions for this run, if any, win over everything below.

Find the gaps in what the doc says its authors need. Ask about what it must do: who uses it and what each of them does, the steps of each flow, the business rules and their exceptions, the data they have or must keep, what is in and out of scope, and what finished looks like. Ask about how well it must do it, in their terms: how many people use it and when, how fast it must feel, when it must be available, who may see what, devices and languages, accessibility, the systems it must work with, legal or industry rules, deadlines, budget, and what matters most. Never ask how to build it (hosting, databases, frameworks, architecture); those are the developer''s, so put anything technical left unclear in your reply.

Do not ask what an earlier round answered, and do not repeat a question still waiting for an answer. Ask a skipped question once more only if it still matters.

If gaps remain, post one round with `doc_update` `questions`: three to six, the biggest unknowns first, grouped by the doc''s sections. Write each in plain words for someone non-technical, with two to four concrete options that each stand on their own: the one most projects pick first and labelled "(Suggested)", no other option labelled, multi-select only where the options can be combined, and a one-line why that says why it matters to them. If the last round has "Anything else?" text, send `anything_else_reply` with it: one plain line that answers it, or says which of this round''s questions follow it up; never name a round number in it. Never use your question tool: post the round and end your turn.

Before you call it done, check that every area above was either asked about or is already answered in the doc; anything you set aside in an earlier reply still needs asking, unless the developer''s instructions dropped it.

If the developer could turn the doc into tickets without asking its authors anything more, there are no gaps left: send `doc_update` with `no_gaps` and the whole new body. Keep the authors'' headings and their own sentences wherever they still hold, weave each answer into the section it belongs to, add a section only where nothing fits, and never mention questions or rounds. Add only what an answer says: never invent details, figures, examples, or requirements the authors did not give. Only questions that were asked and skipped twice, and still matter, go under a short "Open points" section.

Never mention AI, an agent, or yourself in anything the authors see. Reply to the developer with what you asked and why, or that the doc is complete and what changed, plus any technical questions for them.';

DELETE FROM instance_templates WHERE kind = 'play_instructions' AND key = 'clarify' AND body = 'Clarify this doc for the people who wrote it. Read it with `doc_get`: its body and its clarification, every earlier round with its questions, answers, skipped questions, and "Anything else?" text. The instructions for this run, if any, win over everything below.

Find the gaps in what the doc says its authors need. Ask about what it must do: who uses it and what each of them does, the steps of each flow, the business rules and their exceptions, the data they have or must keep, what is in and out of scope, and what finished looks like. Ask about how well it must do it, in their terms: how many people use it and when, how fast it must feel, when it must be available, who may see what, devices and languages, accessibility, the systems it must work with, legal or industry rules, deadlines, budget, and what matters most. Never ask how to build it (hosting, databases, frameworks, architecture); those are the developer''s, so put anything technical left unclear in your reply.

Do not ask what an earlier round answered, and do not repeat a question still waiting for an answer. Ask a skipped question once more only if it still matters.

If gaps remain, post one round with `doc_update` `questions`: three to six, the biggest unknowns first, grouped by the doc''s sections. Write each in plain words for someone non-technical, with two to four concrete options that each stand on their own: the one most projects pick first and labelled "(Suggested)", no other option labelled, multi-select only where the options can be combined, and a one-line why that says why it matters to them. If the last round has "Anything else?" text, send `anything_else_reply` with it: one plain line that answers it, or says which of this round''s questions follow it up; never name a round number in it. Never use your question tool: post the round and end your turn.

Before you call it done, check that every area above was either asked about or is already answered in the doc; anything you set aside in an earlier reply still needs asking, unless the developer''s instructions dropped it.

If the developer could turn the doc into tickets without asking its authors anything more, there are no gaps left: send `doc_update` with `no_gaps` and the whole new body. Keep the authors'' headings and their own sentences wherever they still hold, weave each answer into the section it belongs to, add a section only where nothing fits, and never mention questions or rounds. Add only what an answer says: never invent details, figures, examples, or requirements the authors did not give. Only questions that were asked and skipped twice, and still matter, go under a short "Open points" section.

Never mention AI, an agent, or yourself in anything the authors see. Reply to the developer with what you asked and why, or that the doc is complete and what changed, plus any technical questions for them.';

UPDATE plays SET instructions = 'Write the answers to this doc''s questions into the doc. Read it with `doc_get`: its body, its clarification and every round. The instructions for this run, if any, win over everything below.

Ask nothing: send no `questions` and use no question tool. If no question has an answer and no round has "Anything else?" text, change nothing and reply that there is nothing to write yet.

Otherwise send `doc_update` with `no_gaps` and the whole new body. Keep the authors'' headings and their own sentences wherever they still hold, put each answer and each "Anything else?" text in the section it belongs to, adding a section only where nothing fits, and change only what a newer or changed answer says. Add only what an answer says, with no invented detail, figure or requirement. Because this run asks nothing more, every question skipped or still unanswered that still matters goes under a short "Open points" section. Never mention questions or rounds in the doc. If the previous round has "Anything else?" text with no reply, send `anything_else_reply` with it: one plain line saying where the doc now covers it, with no round number.

Never mention AI, an agent, or yourself in anything the authors see. Reply to the developer with what changed in the doc, which points are still open, and any technical questions for them.'
WHERE builtin_key = 'update-doc' AND instructions = 'Write the answers to this doc''s questions into the doc, for the people who wrote it. Read it with `doc_get`: its body and its clarification, every round with its questions, answers, skipped questions, and "Anything else?" text. The instructions for this run, if any, win over everything below.

Ask nothing: never send `questions` and never use your question tool. If no question has an answer and no round has "Anything else?" text, change nothing and reply that there is nothing to write yet.

Otherwise send `doc_update` with `no_gaps` and the whole new body. Keep the authors'' headings and their own sentences wherever they still hold, weave each answer and each "Anything else?" text into the section it belongs to, add a section only where nothing fits, and never mention questions or rounds. The body may already hold earlier answers: change only what a newer or changed answer says. Add only what an answer says: never invent details, figures, examples, or requirements the authors did not give. Questions skipped or still unanswered that still matter go under a short "Open points" section. If the last round has "Anything else?" text with no reply, send `anything_else_reply` with it: one plain line that says where the doc now covers it; never name a round number in it.

Never mention AI, an agent, or yourself in anything the authors see. Reply to the developer with what changed in the doc and which points are still open, plus any technical questions for them.';

DELETE FROM instance_templates WHERE kind = 'play_instructions' AND key = 'update-doc' AND body = 'Write the answers to this doc''s questions into the doc, for the people who wrote it. Read it with `doc_get`: its body and its clarification, every round with its questions, answers, skipped questions, and "Anything else?" text. The instructions for this run, if any, win over everything below.

Ask nothing: never send `questions` and never use your question tool. If no question has an answer and no round has "Anything else?" text, change nothing and reply that there is nothing to write yet.

Otherwise send `doc_update` with `no_gaps` and the whole new body. Keep the authors'' headings and their own sentences wherever they still hold, weave each answer and each "Anything else?" text into the section it belongs to, add a section only where nothing fits, and never mention questions or rounds. The body may already hold earlier answers: change only what a newer or changed answer says. Add only what an answer says: never invent details, figures, examples, or requirements the authors did not give. Questions skipped or still unanswered that still matter go under a short "Open points" section. If the last round has "Anything else?" text with no reply, send `anything_else_reply` with it: one plain line that says where the doc now covers it; never name a round number in it.

Never mention AI, an agent, or yourself in anything the authors see. Reply to the developer with what changed in the doc and which points are still open, plus any technical questions for them.';

UPDATE plays SET description = 'Builds the ticket on its own branch, proves it, and opens a pull request; the project''s memories say how it finishes.' WHERE builtin_key = 'fix-with-ai' AND description = 'Reads the ticket, implements a fix on its own branch, and opens a pull request.';

UPDATE plays SET description = 'Splits a doc into thin, ordered tickets, each with acceptance criteria, how to verify it, and its blockers.' WHERE builtin_key = 'to-tickets-via-ai' AND description = 'Splits a doc into tickets a developer could pick up independently.';

UPDATE ticket_types SET body_template = '## What needs doing


## Acceptance criteria


## How to verify


## Out of scope

'
WHERE name = 'task' AND body_template = '## What needs doing


## Acceptance criteria

';

DELETE FROM instance_templates WHERE kind = 'ticket_body' AND key = 'task' AND body = '## What needs doing


## Acceptance criteria

';

UPDATE ticket_types SET body_template = '## Steps to reproduce


## Expected result


## Actual result


## Screenshots

'
WHERE name = 'bug' AND body_template = '## Steps to reproduce


## Expected result


## Actual result


## Provide screenshot

';

DELETE FROM instance_templates WHERE kind = 'ticket_body' AND key = 'bug' AND body = '## Steps to reproduce


## Expected result


## Actual result


## Provide screenshot

';

UPDATE ticket_types SET body_template = '## Why


## Acceptance criteria


## How to verify


## Out of scope

'
WHERE name = 'feature' AND body_template = '## Why


## Acceptance criteria


## Out of scope

';

DELETE FROM instance_templates WHERE kind = 'ticket_body' AND key = 'feature' AND body = '## Why


## Acceptance criteria


## Out of scope

';

DELETE FROM instance_templates WHERE kind = 'interview' AND key = '' AND body = '## What languages and frameworks does this project use?
Name versions only where they are pinned on purpose.

## How is the code organised?
- Layers (routes, logic, storage)
- Feature folders
- Entities and systems
- Functional core with a thin outer shell

## How do errors travel?
And what gets logged, at which level.
- Returned and wrapped
- Thrown and caught at the edge
- Result types

## When are tests written?
- Before the code
- With the change
- Only for bugs
- No tests yet

## Which tests does a change need?
And the coverage floor, if any.
- [ ] Unit
- [ ] Integration against real dependencies
- [ ] End-to-end in this repo
- [ ] End-to-end in a separate repo

## Which style rules matter most?
Skip what a linter already enforces.
- [ ] Early return, no else
- [ ] Small functions
- [ ] Comments only for why
- [ ] Strict types, no any

## When may a change add a dependency?
- Freely
- When it saves real code
- Only after asking
- Only with a written decision

## Where do secrets live?
And what must never be committed or logged.
- Environment variables
- A secrets manager
- An encrypted file in the repo

## How does a change reach the main branch?
And how commit messages are written.
- Pull request, squash merge
- Pull request, merge commit
- Straight to main

## Where are decisions written down?
- Decision records in the repo
- A docs folder
- A wiki outside the repo
- Nowhere yet

## Does this project have a user interface?
If yes, the design system, screen sizes, and accessibility rules.
- Web
- Mobile
- Both
- None

## Which words mean something specific here?
One per line, the term then what it means.
';

DELETE FROM interview_templates WHERE body = '## What languages and frameworks does this project use?
Name versions only where they are pinned on purpose.

## How is the code organised?
- Layers (routes, logic, storage)
- Feature folders
- Entities and systems
- Functional core with a thin outer shell

## How do errors travel?
And what gets logged, at which level.
- Returned and wrapped
- Thrown and caught at the edge
- Result types

## When are tests written?
- Before the code
- With the change
- Only for bugs
- No tests yet

## Which tests does a change need?
And the coverage floor, if any.
- [ ] Unit
- [ ] Integration against real dependencies
- [ ] End-to-end in this repo
- [ ] End-to-end in a separate repo

## Which style rules matter most?
Skip what a linter already enforces.
- [ ] Early return, no else
- [ ] Small functions
- [ ] Comments only for why
- [ ] Strict types, no any

## When may a change add a dependency?
- Freely
- When it saves real code
- Only after asking
- Only with a written decision

## Where do secrets live?
And what must never be committed or logged.
- Environment variables
- A secrets manager
- An encrypted file in the repo

## How does a change reach the main branch?
And how commit messages are written.
- Pull request, squash merge
- Pull request, merge commit
- Straight to main

## Where are decisions written down?
- Decision records in the repo
- A docs folder
- A wiki outside the repo
- Nowhere yet

## Does this project have a user interface?
If yes, the design system, screen sizes, and accessibility rules.
- Web
- Mobile
- Both
- None

## Which words mean something specific here?
One per line, the term then what it means.
';

DELETE FROM instance_templates WHERE kind = 'agent_prompt' AND key = 'footer' AND body = 'Standing rules:
- A ticket''s body is its spec: change it only when a person asks. Put anything lasting you learn in a note with message_post''s file, and if the spec looks wrong, say so and suggest the person edit it.
- Memories are the team''s shared notes for agents, one project each. Save a durable fact worth keeping with memory_create, or memory_update the one that already covers it, passing the project''s project_id; always save one when someone says "remember X". Find others with memory_list and read them with memory_get when a task needs them.';
