---
title: Plays
description: Configure and run workspace plays that start an Agent turn from a ticket, a document, or a project's interview.
sidebar:
  order: 10
---

A **Play** is a workspace-scoped, pre-configured Agent turn fired by a person.
It is not an Automation. A play runs on that person's paired Harness and posts
its conversation into the thread of its target: a ticket, a document, or a
project's interview.

## Configure a play

Open **Configuration → Plays**. Every workspace starts with six ordinary plays:

- **Fix with AI** is a ticket play shown in the In progress stage.
- **To tickets via AI** is a document play.
- **Interview** is an interview play, run from a project's Interview page.
- **Test with AI** is a ticket play shown in the Testing stage.
- **Draft interview** is an interview play that drafts answers from the
  interview's follow sources.
- **Audit via AI** is an interview play, shown beside the memory once the
  interview memory exists. It reads the project's sources under question (a
  predecessor's code), or the project's own checkout when there are none, and
  writes one doc in the Main folder, "Audit of <what>, <date>", with a
  verdict, a `path:line`, and the rule for each finding under the memory's
  headings. A finished run links its doc on the Interview page.

Each keeps a built-in key (`fix-with-ai`, `to-tickets-via-ai`, `interview`,
`test-with-ai`, `interview-draft`, `audit`) through renames. A new workspace's copies take their
instructions from the instance's templates (see
[Templates](/docs/guide/memories/#templates)); editing those never rewrites a
workspace that already exists, and resetting a built-in play's instructions
gives it the instance's current text.

The same screen can create, edit, exclude users from, and delete plays. A play
has these fields:

- **Label**, the button text.
- **Type**, **Ticket**, **Doc**, or **Interview**. It cannot change after
  creation.
- **Show when**, exactly one board stage for a ticket play. Other plays have
  no stage.
- **Description**, shown as the button tooltip.
- **Instructions**, sent as the play's base instructions.
- **Enabled** and **Excluded projects**.

The five possible stages are Backlog, In progress, Review, Testing, and Done.
An enabled play is offered only when its type, project exclusions, and stage
match the current target.

## Run a play

Use the play button on a ticket or document. The run dialog can choose:

- memories for the Agent to read first;
- custom instructions;
- footer memories, which the Agent reads once the work is done to conclude
  the run, such as deciding which column the ticket now belongs in;
- a paired computer, provider, and model.

A play never moves the ticket by itself. Where the ticket goes is the Agent's
call, guided by the footer memories picked for the run.

The memory selection starts with the starter's latest
choices for this play and project. Custom instructions start blank. The
harness choice also reuses the starter's latest choice; if there is no prior
choice, it falls back to the resolved project or user pairing choice. A run
needs a usable computer. If no computer is paired, expired, offline, missing
a T3 project, or ambiguous because no default was chosen, the UI shows the
reason instead of firing a turn.

A ticket play also tells the Agent about the ticket's links. On a bug it is
told which ticket the bug was found in and reads that ticket's body, doc, and
pull requests itself with `ticket_get`, one hop only, never the origin's own
origin.
A bug filed with its origin unknown is run with that said plainly. A play on a
blocked ticket asks "are you sure?" before it runs, and the Agent is told each
blocker and whether it is done.

A doc play locks its doc as the run starts, so the doc's title and body
refuse edits from people and agents alike, and leaves it locked when the run
ends. Starting the play needs no `docs:lock`; someone holding it unlocks the
doc when it should change again. A doc that was already locked stays as it
was. The trail and the doc's thread say the run locked it. A custom doc play
meant to edit its own doc is refused by that lock; the built-in **To tickets
via AI** only reads its doc.

Each press creates a persisted **Trail**. It records the starter, target,
selected memories, instructions, resolved Harness choice, structured activity,
the Agent reply, and the outcome. The trail is visible from the target and its
conversation thread.

Trail states are `starting`, `running`, `waiting`, `done`, `failed`, and
`interrupted`. `waiting` means the Agent asked a question. The starter can
answer it and the same trail continues. The starter or a `plays:write` holder
can stop a run, including one in `waiting`. Fifteen minutes without Harness
activity fails the run. If the connection to the computer drops mid-run, the
trail shows **Reconnecting to T3 Code…** and Nexul keeps redialing for five
minutes; once it is back, the steps taken while it was away land in order and
the run ends as the Agent ended it. Only a connection that stays down past
those five minutes fails the run. A trail keeps the newest 300 activity
entries.

The run's prompt names its context instead of carrying it: the play and its
instructions, the ticket or doc by key or id and title for the Agent to read
with `ticket_get` or `doc_get`, then the interview memory, the other
always-included memories, and the ones picked for the run, each with its id,
for the Agent to read with `memory_get` before it starts, and last your custom
instructions. It carries no memory body and none of the thread's earlier
messages, so memory size never stops a run. The "Started" message still lands
in the thread. Images embedded in the ticket or doc body travel with the turn,
limited to 10 MiB each and 25 MiB in total; an oversized or non-image
attachment is left out.

## The Interview play

The Interview play runs on a project's Interview page
(`/<workspace>/projects/<prefix>/interview`), from **Done** once every question
is answered or skipped, or **Regenerate** once the project's interview memory
exists. Its target is the project: `target_type` is `interview` and
`target_id` is the project id. The run posts into the project's interview
thread, a conversation of its own that no page shows; the page shows the run's
state from its trail.

The Agent is told the project's name and id and the answers the project
already records, such as where its tests live from the project wizard. It
opens the interview memory with `memory_create` and `kind` `interview`, which creates it
empty the first time and returns the workspace's Interview template questions
and the project's stored answers and sources. It reads the checkout and every
source whose stance is question, such as an earlier phase's code, then asks
about skipped questions, gaps, anything the code contradicts, and what a
question source did that the answers do not settle, a round of questions at a
time, each with its recommended answer as the first option and a line on why
it is asked that names what was found and where. Follow sources are read only
as context; drafting from them is the Draft interview play's. The run's prompt
names each project source under question with its checkout path when the
starter's project link for it is on the run's computer. Each answered round is
stored with the project's answers. It records nothing from the code or a
source that was not confirmed, then writes
the interview memory with `memory_update` as rules rather than a transcript,
under the 8,000-character cap, keeping every existing rule no answer
contradicts. Memory versioning makes any change revertible.

The last step of the project wizard offers the interview while the project has
none. Skipping it, or leaving the wizard without starting it, asks "are you
sure?" and says what agents lose without it. A project without an interview is
never blocked: its board shows a banner linking to the Interview page until
the interview memory exists, which can be dismissed for the browser session.

## The Draft interview play

Draft interview runs on a project's interview like the Interview play, in the
same hidden thread, and never asks anything. It reads the interview memory
with its questions, answers, sources, and drafts, then every source whose
stance is follow, reading a large one selectively. It drafts each template
question with no answer or a skip that a source speaks to, and an answered
question only where the sources now disagree with the answer, which the page
shows as a suggested change. Each draft names its source ids and one line on
where it came from, and is saved with `memory_update` and the run's trail id
as it is found. Sources under question are left to the Interview play.

The run's prompt names its trail id and, for each project source, that
project's checkout path when the starter's project link for it is on the
run's computer, or says there is no checkout there. As the run starts, the
project's suggested changes are cleared, so only those the run drafts again
come back; drafts on unanswered questions stay until the run replaces them.

## The Test with AI play

Test with AI is offered on a ticket in a Testing column. It tests the ticket
the way a person pressing **Pass** or **Fail** in the Test this panel would.
The Agent follows the testing strategy in the project's interview memory. It
reads the ticket's acceptance criteria and where to test from `ticket_get`,
which is never production. When there is no safe test environment
it stops without a result and says a deploy branch on its own network is
needed. Otherwise it checks the live URL against each criterion and runs the
project's tests, from the tests repository when the project has one. Where the
interview calls for an automated end-to-end suite, it also adds or extends a
test covering the criteria.

It then records the result with `ticket_test_report`, whose `pass` and `fail`
outcomes do what the panel's buttons do, signed "Nexul · from" the person who ran
the play. A pass posts "Passed by Nexul · from <login>" and moves the card to
the first Done column. A fail posts the bug template (steps to reproduce,
expected result, actual result) under "Test failed by Nexul · from <login>" to
the ticket's thread and moves the card back to In progress. Leave out footer memories that move the card for this play,
because the result already moves it.

## The decisions check

When a ticket enters a column in the Done stage, and the workspace has the
check switched on, Nexul fires the built-in **Decisions check** once, with no
button. The switch is on the Automations page, among the default automations,
and starts off. It runs like a play started by the
person who moved the card, on their paired computer. When an automation moved
it, as the default automation does once the ticket's pull requests merge, it
runs on the ticket's developer's computer instead. The Agent reads the
ticket, its pull requests, and the project's decisions log (see Memories),
then adds an entry, marks an older one superseded, or leaves the log alone.
Its run is an ordinary trail labelled Decisions check. Moving a done ticket
between two Done columns does not fire it again.

It never fails silently. If the run cannot start, because nobody paired a
computer, the provider is not set up there, the computer is offline, the
ticket has no developer, or another run holds the ticket, the failed trail
stays on the ticket and the ticket page shows **Decisions check didn't run**
with the reason and a **Run check** button. Pressing it runs the check on your
own computer. Agents retry with `play_run` and `decisions_check: true`. Either
way it needs `plays:run`.

## Permissions

`plays:read` shows the Plays settings section and lets a member list plays.
`plays:write` creates and edits plays and manages the per-user **Exclude
users** grants. `plays:delete` deletes them. `plays:run` controls who may fire
a specific play. A deny overwrite for `plays:run` on a play excludes that user
even when their role can otherwise run it.

The HTTP definition routes are under
`/api/workspaces/{workspaceID}/plays`. Runs use
`/api/plays/{playID}/run`, `/api/plays/runs/{id}`,
`/api/plays/runs/{id}/stop`, and `/api/plays/runs/{id}/answer`; the list of a
target's runs is `/api/plays/runs?target_type=&target_id=`. The decisions
check retries through `POST /api/plays/decisions-check` with a `ticket_id`.
MCP exposes the matching `play_*` tools and `trail_list` and `trail_update` for
runs; `play_run` with `decisions_check: true` reruns the decisions check.
`play_run` and `trail_list` take `ticket`, `doc`, or `interview` as
`target_type`. The run events `play.run_started`,
`play.run_waiting`, and `play.run_finished` carry the same `target_type`, with
the project's name as `target_title` for an interview. The project's interview
thread is `POST /api/chat/projects/{projectID}/interview-thread`, or
`message_list` and `message_post` with the `project_id` over MCP.
