---
title: Plays
description: Press a play on a ticket, a doc, or a project's interview and an agent on your own computer does the work, with every step recorded.
sidebar:
  order: 10
---

A play is a button that starts an agent on a piece of work, such as **Fix with AI** on a ticket. The agent runs on your own paired computer, with your permissions, and its conversation lands in the ticket's or doc's thread. Before your first run, [pair a computer](/docs/guide/paired-computers/) and [set it up](/docs/guide/computer-setup/).

## Run a play

1. Open a ticket, a doc, or a project's **Interview** page and press the play's button. A ticket play only shows while the ticket sits in the stage it belongs to.
2. In the run dialog, pick the memories the agent should read first under **Main**, and the ones it reads at the end under **Footer**. The dialog starts with what you picked last time for this play and project.
3. Add **Instructions for this run** if you want to steer it. They win over the play's own instructions.
4. Check the computer, provider, and model. They come from your project link or your defaults; change them here for this run only.
5. Start the run.

If the play can't run, the dialog says why instead: no computer paired, a pairing that expired, the computer offline, no T3 project for this project, or several computers and no default. Each message names the setting that fixes it. A play on a blocked ticket asks you to confirm first.

The agent decides where the ticket goes when it finishes. It reads your footer memories for that, so a footer memory that says "move the card to Review when the pull request is open" is how you get cards to move.

## Follow a run

Every press leaves a trail, listed in the **Trail** section of the ticket, doc, or Interview page. Open one to see who started it, what was picked, and a transcript of what the agent said and did: each command, file change, and MCP call as its own row.

While a run is going:

- **The agent asks you something.** A question card shows on the trail and in the thread. Answer it and the same run carries on. You can answer in T3 Code instead, picking an option or typing your own; the card then shows that answer.
- **You want it to stop.** Press stop on the play's button. You, or anyone who manages plays, can stop it, even while it waits on a question.
- **The computer drops off.** The trail shows **Reconnecting to T3 Code…** and Nexul keeps trying. The run fails only if the computer stays unreachable.
- **Nothing happens for 15 minutes.** The run fails and its turn in T3 Code is stopped.

When a run has ended, type into **Continue this run** to send the agent its next step on the same thread. The play's instructions are not sent again. If you keep working on the thread in T3 Code itself, the trail picks up those turns while you have Nexul open.

A doc play locks its doc when it starts, so nobody edits the doc under the agent, and the doc stays locked afterwards. Someone with the lock permission unlocks it. **Clarify via AI** is the exception: it unlocks the doc when its round ends.

## The built-in plays

Every workspace starts with seven. They are ordinary plays: rename, edit, or delete them.

| Play | Runs on | What it does |
|---|---|---|
| **Fix with AI** | a ticket in an In progress column | Reads the ticket, fixes it on its own branch, and opens a pull request. |
| **Test with AI** | a ticket in a Testing column | Tests the ticket against its acceptance criteria, then passes or fails it. |
| **To tickets via AI** | a doc | Splits the doc into tickets a developer could pick up on their own. |
| **Clarify via AI** | a doc | Asks the doc's authors about what it leaves open, a round at a time. See [Clarify a doc](/docs/guide/docs-tickets-and-board/#clarify-a-doc). |
| **Interview** | a project's Interview page | Asks follow-ups and writes the project's rules for agents. See [Interview](/docs/guide/interview/). |
| **Draft interview** | a project's Interview page | Drafts interview answers from the project's sources. |
| **Audit via AI** | a project's Interview page | Checks code against the interview memory and writes the findings as a doc. |

**Test with AI** works like pressing **Pass** or **Fail** yourself. It never tests production: with no safe test environment it stops and says a branch deploy is needed. A pass moves the card to the first Done column; a fail posts steps to reproduce, the expected result, and the actual result to the thread and moves the card back to In progress. Leave out footer memories that move the card for this play, because the result already does.

On a bug, a ticket play tells the agent which ticket the bug was found in, so it can read that ticket and its pull requests.

## Create or change a play

Open **Configuration → Plays**. A play has:

- **Label**, the button text. Each play in a workspace has its own label, ignoring case, since an automation names a play by it; a label already in use is refused at the field.
- **Type**: ticket, doc, or interview. It can't change later.
- **Show when**, the one board stage a ticket play shows in.
- **Description**, shown as the button's tooltip.
- **Instructions**, what the agent should do on every run.
- **Enabled**, and **Excluded projects** where it never shows.

A built-in play says whether its instructions match the instance's template, with **Reset to instance template** when they don't. See [Templates](/docs/guide/memories/#templates).

To stop one person from running one play, press **Exclude users** on its row.

## Auto plays

An auto play starts its play by itself when something happens to a ticket or doc, so nobody has to press the button. A play can have several. Open the play in **Configuration → Plays**, press **Edit**, and go to the **Auto plays** tab. Interview plays have none.

Each auto play reads as one sentence, such as "When a ticket **becomes unblocked**, if **Type is Bug** → **High**, else **Normal**, runs on **Developer**", with a switch to turn it on or off. A new one starts switched off. Press a row to change it, or use its menu to duplicate or delete it.

An auto play has:

- **When**, the moment it waits for. A ticket play: the ticket becomes unblocked, enters a stage, is created, gets a developer, gets a tester, or fails a test. A doc play: the doc is created, or changes. A change counts once edits have stopped for 10 minutes, and never for an agent's edits.
- **If**, conditions on the ticket's or doc's fields: type, project, stage, status column, category, label, developer, tester, whether it has a source doc or a linked pull request, and whether it is blocked; a doc's project and folder. Pick whether all or any of them must match, and add a group one level in for an "any of these" inside an "all of these". With no conditions, every match of the moment runs it.
- **Priority**, High, Normal, or Low, with rules such as "High if Type is Bug" checked in order, then the level for everything else. A higher priority runs first in the person's queue.
- **Limits**, at most once per ticket or doc every hour, 24 hours, or 7 days, or no limit.
- **Run on**, whose computer runs it: the ticket's developer, its tester, or whoever caused the moment. A doc play always runs on whoever caused it.

The play's **Show when** stage does not apply to an auto play; add a **Stage is** condition instead. A run waits until its person has a free slot and their computer is online, and its conditions are checked again before it starts. The person it runs on needs `plays:run`.

**Each ticket runs at most 5 auto plays a day** across every play, so two auto plays can't keep starting each other. Change the number under the list of any ticket play's auto plays; it applies to the whole workspace.

Under the list, "Right now" says how many runs of the play are queued and whom they wait on, such as "1 waiting on Alice (computer offline)".

On the ticket, the rail's **Plays** section shows what is waiting: each queued run with its person and why it waits (computer offline, no free slot, ticket busy, or paused), and **Cancel** for that person or anyone with `autoplays:write`. Once a ticket reaches the daily cap, it shows **Auto plays paused** with the runs today; the ticket's developer or anyone with `autoplays:write` can press **Resume**, or it resumes by itself as the day rolls on. What already happened sits in the ticket's thread at the time it happened: "Fix with AI skipped: no longer unblocked" when the ticket stopped matching before the run's turn came, or "Fix with AI didn't run" with the reason when there was nobody to run it on. **Run it** on the latest one runs the play on your own computer. A doc shows the same: queued and paused beside its trail, skipped and didn't run in its thread.

## The decisions check

The decisions check is a play nobody presses. When a ticket enters a Done column, the agent reads the ticket and its pull requests and updates the project's [decisions log](/docs/guide/memories/#the-decisions-log): it adds an entry, marks an older one superseded, or leaves the log alone.

It starts off. Switch it on per workspace on the **Automations** page, among the default automations. It runs on the computer of the person who moved the card, or the ticket's developer's when an automation moved it.

If it can't start, the ticket shows **Decisions check didn't run** with the reason. Press **Run check** to run it on your own computer.

## Who can do what

| Permission | Lets you |
|---|---|
| `plays:run` | see and press plays |
| `plays:read` | open **Configuration → Plays** |
| `plays:write` | create and edit plays, exclude people, stop anyone's run |
| `plays:delete` | delete plays |
| `autoplays:read` | see a play's auto plays and the daily cap |
| `autoplays:write` | add, change, and switch auto plays, and set the daily cap |
| `autoplays:delete` | delete auto plays |

Agents run plays with `play_run` and read trails with `trail_list`. See [MCP server](/docs/guide/mcp-server/).
