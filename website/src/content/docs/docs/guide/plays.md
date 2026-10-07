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

- **The agent asks you something.** A question card shows on the trail and in the thread. Answer it and the same run carries on.
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

- **Label**, the button text.
- **Type**: ticket, doc, or interview. It can't change later.
- **Show when**, the one board stage a ticket play shows in.
- **Description**, shown as the button's tooltip.
- **Instructions**, what the agent should do on every run.
- **Enabled**, and **Excluded projects** where it never shows.

A built-in play says whether its instructions match the instance's template, with **Reset to instance template** when they don't. See [Templates](/docs/guide/memories/#templates).

To stop one person from running one play, press **Exclude users** on its row.

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

Agents run plays with `play_run` and read trails with `trail_list`. See [MCP server](/docs/guide/mcp-server/).
