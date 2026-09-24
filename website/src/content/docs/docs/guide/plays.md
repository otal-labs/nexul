---
title: Plays
description: Configure and run workspace plays that start an Agent turn from a ticket or document.
sidebar:
  order: 10
---

A **Play** is a workspace-scoped, pre-configured Agent turn fired by a person.
It is not an Automation. A play runs on that person's paired Harness and posts
its conversation into the ticket or document thread.

## Configure a play

Open **Settings → Plays**. Every workspace starts with two ordinary plays:

- **Fix with AI** is a ticket play shown in the In progress stage.
- **To tickets via AI** is a document play.

The same screen can create, edit, exclude users from, and delete plays. A play
has these fields:

- **Label**, the button text.
- **Type**, either **Ticket** or **Doc**. It cannot change after creation.
- **Show when**, exactly one board stage for a ticket play. A doc play has no
  stage.
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

## Permissions

`plays:read` shows the Plays settings section and lets a member list plays.
`plays:write` creates and edits plays and manages the per-user **Exclude
users** grants. `plays:delete` deletes them. `plays:run` controls who may fire
a specific play. A deny overwrite for `plays:run` on a play excludes that user
even when their role can otherwise run it.

The HTTP definition routes are under
`/api/workspaces/{workspaceID}/plays`. Runs use
`/api/plays/{playID}/run`, `/api/plays/runs/{id}`,
`/api/plays/runs/{id}/stop`, and `/api/plays/runs/{id}/answer`. MCP exposes
the matching `play_*` and `play_run_*` tools.
