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

Open **Settings → Plays**. Every workspace starts with three ordinary plays:

- **Fix with AI** is a ticket play shown in the In progress stage.
- **To tickets via AI** is a document play.
- **Interview** is an interview play, run from a project's Interview page.

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

- memories to inline;
- custom instructions;
- the ticket column to move to on success;
- a paired computer, provider, and model.

The memory selection and success column start with the starter's latest
choices for this play and project. Custom instructions start blank. The
harness choice also reuses the starter's latest choice; if there is no prior
choice, it falls back to the resolved project or user pairing choice. A run
needs a usable computer. If no computer is paired, expired, offline, missing
a T3 project, or ambiguous because no default was chosen, the UI shows the
reason instead of firing a turn.

A ticket play also tells the Agent about the ticket's links. On a bug it
receives one hop of origin context: the body of the ticket the bug was found
in, that ticket's doc, and its pull requests, never the origin's own origin.
A bug filed with its origin unknown is run with that said plainly. A play on a
blocked ticket asks "are you sure?" before it runs, and the Agent is told each
blocker and whether it is done.

Each press creates a persisted **Trail**. It records the starter, target,
selected memories, instructions, resolved Harness choice, structured activity,
the Agent reply, and the outcome. The trail is visible from the target and its
conversation thread.

Trail states are `starting`, `running`, `waiting`, `done`, `failed`, and
`interrupted`. `waiting` means the Agent asked a question. The starter can
answer it and the same trail continues. The starter or a `plays:write` holder
can stop a run, including one in `waiting`. Fifteen minutes without Harness
activity fails the run. A trail keeps the newest 300 activity entries.

Selected memories must fit the run limits: 20,000 characters per memory and
60,000 characters for the selected memories together. Images passed through a
turn are limited to 10 MiB each and 25 MiB in total. An oversized or
non-image attachment is recorded as omitted.

## The Interview play

The Interview play runs on a project's Interview page
(`/projects/<prefix>/interview`), where the button reads **Run the interview**,
or **Re-run the interview** once the project's interview memory exists. Its
target is the project: `target_type` is `interview` and `target_id` is the
project id. The run posts into the project's interview thread, a conversation
of its own that the Interview page shows once a run has created it.

The Agent is told the project's name and id and the answers the project
already records, such as where its tests live from the project wizard. It
opens the interview memory with `memory_create_interview`, which copies the
workspace's Interview template the first time. It asks one question at a time,
each with its recommended answer as the first option. Its first question asks
whether to scan the codebase for answers; if so, it reads the checkout, then
asks about each finding until it is confirmed or corrected. It saves the
interview memory with `memory_update` after each category, as rules rather
than a transcript, under the 8,000-character cap. A re-run amends the existing
interview instead of starting over, and memory versioning makes any amendment
revertible.

The last step of the project wizard offers the interview while the project has
none. Skipping it, or leaving the wizard without starting it, asks "are you
sure?" and says what agents lose without it. A project without an interview is
never blocked: its board shows a banner linking to the Interview page until
the interview memory exists, which can be dismissed for the browser session.

## The decisions check

When a ticket enters a column in the Done stage, Nexul fires the built-in
**Decisions check** once, with no button. It runs like a play started by the
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
own computer. Agents retry with `decisions_check_run`. Either way it needs
`plays:run`.

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
MCP exposes the matching `play_*` and `play_run_*` tools and
`decisions_check_run`; `play_run` and `play_list_runs` take `ticket`, `doc`,
or `interview` as `target_type`. The run events `play.run_started`,
`play.run_waiting`, and `play.run_finished` carry the same `target_type`, with
the project's name as `target_title` for an interview. The project's interview
thread is `POST /api/chat/projects/{projectID}/interview-thread`, or
`interview_thread_get` over MCP.
