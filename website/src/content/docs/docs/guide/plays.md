---
title: Plays
description: Press a play on a ticket, a doc, or a project's interview, or let one start itself when a service crashes, and an agent on a computer does the work, with every step recorded.
sidebar:
  order: 10
---

A play is a button that starts an agent on a piece of work, such as **Fix with AI** on a ticket. The agent runs on your own paired computer, with your permissions, and its conversation lands in the ticket's or doc's thread. Before your first run, [add a computer](/docs/guide/paired-computers/) and [set it up](/docs/guide/computer-setup/).

## Run a play

1. Open a ticket, a doc, or a project's **Interview** page and press the play's button. A ticket play only shows while the ticket sits in the stage it belongs to.
2. In the run dialog, pick the memories the agent should read first under **Main**, and the ones it reads at the end under **Footer**. The dialog starts with what you picked last time for this play and project.
3. Add **Instructions for this run** if you want to steer it. They win over the play's own instructions.
4. Check **Where to run**. The first time you run a play in a project you haven't linked, the dialog asks for a computer and a T3 project, with your defaults filled in. Your pick is saved as your [project link](/docs/guide/paired-computers/#choose-where-turns-run), so later runs there go straight to it. After that the dialog shows where in one line; **Change** picks somewhere else and saves that instead.
5. Check the provider and model on the pill under it. They come from your project link; change them here for this run only. Your link keeps its own model, even on a first run or a **Change**.
6. Start the run.

A run keeps the target's T3 thread unless that thread is in another T3 project, which starts a fresh one; picking the same place again keeps it. Nexul saves a picked T3 project only once the computer lists it, so a computer that is offline can't take a new place until it is back. Once a run has started it stays where it started: an answer to its question, **Continue this run**, and following it again after a Nexul restart all go to its own thread, even if you change or clear your link meanwhile.

If no computer or T3 project is set as your default, pick them in the run dialog. If the play can't run, the dialog says why instead: no computer paired, a pairing that expired, the computer offline, or setup still needed. A play on a blocked ticket asks you to confirm first.

The agent decides where the ticket goes when it finishes. It reads your footer memories for that, so a footer memory that says "move the card to Review when the pull request is open" is how you get cards to move.

## Follow a run

Every press leaves a trail, listed in the **Trail** section of the ticket, doc, or Interview page. A trail shows to whoever started the run and to everyone who can see the play. Open one to see who started it, what was picked, and a transcript of what the agent said and did. While the agent works, one line names the step it is on, such as "Running bun run test". Each stretch of work then folds to a line such as "Worked for 14s · explored 2 files, ran 1 command". Open the fold to see the files it read or edited, each command it ran, and any other tool it called. Click a file name, a command, or a tool to see its arguments and result.

While a run is going:

- **The agent asks you something.** A question card shows on the trail and in the thread. Answer it and the same run carries on. You can answer in T3 Code instead, picking an option or typing your own; the card then shows that answer.
- **You want it to stop.** Press stop on the play's button. You, or anyone who manages plays, can stop it, even while it waits on a question.
- **The computer drops off.** The trail shows **Reconnecting to T3 Code…** and Nexul keeps trying. The run fails only if the computer stays unreachable.
- **Nothing happens for 15 minutes.** The run fails and its turn in T3 Code is stopped.

When a run has ended, type into **Continue this run** to send the agent its next step on the same thread. The play's instructions are not sent again. If you keep working on the thread in T3 Code itself, the trail picks up those turns while you have Nexul open. If the thread is gone from T3 Code, the play starts again as a new run in the same place, with your message added to its instructions. When Nexul doesn't know where an older run ran, the run dialog opens on **Where to run** instead, with your message filled in.

A doc play locks its doc when it starts, so nobody edits the doc under the agent, and the doc stays locked afterwards. Someone with the lock permission unlocks it. **Clarify via AI** and **Update doc via AI** are the exceptions: they unlock the doc when their run ends.

## The built-in plays

Every workspace starts with ten. They are ordinary plays: rename, edit, or delete them.

| Play | Runs on | What it does |
|---|---|---|
| **Fix with AI** | a ticket in an In progress column | Reads the ticket, fixes it on its own branch, and opens a pull request. |
| **Test with AI** | a ticket in a Testing column | Tests the ticket against its acceptance criteria, then passes or fails it. |
| **To tickets via AI** | a doc | Splits the doc into tickets a developer could pick up on their own. |
| **Clarify via AI** | a doc | Asks the doc's authors about what it leaves open, a round at a time. See [Clarify a doc](/docs/guide/docs-tickets-and-board/#clarify-a-doc). |
| **Update doc via AI** | a doc | Writes the answers to the doc's questions into it, without asking more. See [Clarify a doc](/docs/guide/docs-tickets-and-board/#clarify-a-doc). |
| **Interview** | a project's Interview page | Asks follow-ups and writes the project's rules for agents. See [Interview](/docs/guide/interview/). |
| **Draft interview** | a project's Interview page | Drafts interview answers from the project's sources. |
| **Audit via AI** | a project's Interview page | Checks code against the interview memory and writes the findings as a doc. |
| **Decisions check** | a ticket in a Done column | Records in the project's decisions log whether the ticket changed how the project works. See [The decisions check](#the-decisions-check). |
| **Investigate a crash** | a stack, when one of its services crashes | Reads the service's logs and the stack's recent deploys, finds the cause, and writes it down. It never deploys. See [Investigate a crash](#investigate-a-crash). |

**Test with AI** works like pressing **Pass** or **Fail** yourself. It never tests production: with no safe test environment it stops and says a branch deploy is needed. A pass moves the card to the first Done column; a fail posts steps to reproduce, the expected result, and the actual result to the thread and moves the card back to In progress. Leave out footer memories that move the card for this play, because the result already does.

On a bug, a ticket play tells the agent which ticket the bug was found in, so it can read that ticket and its pull requests.

## Create or change a play

Open **Configuration → Plays**. A play has:

- **Label**, the button text. Each play in a workspace has its own label, ignoring case, since an automation names a play by it; a label already in use is refused at the field. Renaming a play that automations still call by its old label with `runPlay` asks first and names them, since they fail until their code uses the new label; **Rename anyway** saves it.
- **Type**: ticket, doc, interview, or stack. It can't change later.
- **Show when**, the one board stage a ticket play shows in.
- **Description**, shown as the button's tooltip.
- **Instructions**, what the agent should do on every run.
- **Enabled**, and **Excluded projects** where it never shows.

A built-in play says whether its instructions match the instance's template, with **Reset to instance template** when they don't. See [Templates](/docs/guide/memories/#templates).

To choose who may see, run, edit or delete one play, right-click its row, pick **Permissions…**, and set **See this play**, **Run**, **Edit** or **Delete** to **Allow** or **Deny** for a role or a person; see [Permissions on a project, a folder, a doc or a play](/docs/guide/people-and-access/#permissions-on-a-project-a-folder-a-doc-or-a-play). The list shows how many a play is hidden from. Running a play needs seeing it: for a role or person denied **See this play**, the **Run** row keeps its own answer and adds a line such as "Can't run while See this play is denied · Denied on play Deploy prod for role Client".

A play someone can't see is gone for them: from **Configuration → Plays**, the buttons and menus on tickets, docs and the Interview page, its auto plays and queued runs, and agents' `play_list`, and its link is not found. Its auto plays never run on them. Runs it already made stay readable to whoever started them and to everyone who can see the play.

## Auto plays

An auto play starts its play by itself when something happens to a ticket, a doc, or a stack, so nobody has to press the button. A play can have several. Open the play in **Configuration → Plays**, press **Edit**, and go to the **Auto plays** tab. Interview plays have none.

Each auto play reads as one sentence, such as "When a ticket **becomes unblocked**, if **Type is Bug** → **High**, else **Normal**, runs on **Developer**", with a switch to turn it on or off. A new one starts switched off. Press a row to change it, or use its menu to duplicate or delete it. Switching one off, or deleting it, skips the runs it still has queued.

An auto play has:

- **When**, the moment it waits for. A ticket play: the ticket becomes unblocked, enters a stage, is created, gets a developer, gets a tester, or fails a test. A doc play: the doc is created, or changes. A change counts once edits have stopped for 10 minutes, and never for an agent's edits. A stack play: a service crashes, recovers once it has run steadily again, or the stack's share of 5xx answers goes over its [error-rate rule](/docs/guide/stacks-and-deploys/#error-rate). The last names no service, so a condition on the service never matches it, and the run is told the stack and reads its open episode with `stack_get`.
- **If**, conditions on the ticket's or doc's fields: type, project, stage, status column, category, label, developer, tester, whether it has a source doc or a linked pull request, and whether it is blocked; a doc's project and folder; a stack's project, the stack, and the service by name. Pick whether all or any of them must match, and add a group one level in for an "any of these" inside an "all of these". With no conditions, every match of the moment runs it.
- **Priority**, High, Normal, or Low, with rules such as "High if Type is Bug" checked in order, then the level for everything else. A higher priority runs first in the person's queue.
- **Limits**, at most once per ticket or doc every hour, 24 hours, or 7 days, or no limit.
- **Run on**, whose computer runs it: the ticket's developer, its tester, whoever caused the moment, or a chosen computer. A doc play has no developer or tester, so it offers whoever caused it and a chosen computer. A stack play offers only a chosen computer. See [Run on a chosen computer](#run-on-a-chosen-computer).

The play's **Show when** stage does not apply to an auto play; add a **Stage is** condition instead. A run waits until its person has a free slot and their computer is online and set up, and its conditions are checked again before it starts. If nothing is paired yet, the computer needs pairing again, no T3 project is linked for the project, the computer's setup isn't done, or several computers are paired and none is the default, the run waits rather than failing and starts once that is fixed. The person it runs on needs to see the play and hold `plays:run`; if they can't see it, it doesn't run, and says why only to those who can.

**Each ticket runs at most 5 auto plays a day** across every play, so two auto plays can't keep starting each other. Change the number under the list of any ticket play's auto plays; it applies to the whole workspace.

Under the list, "Right now" says how many runs of the play are queued and whom they wait on, such as "1 waiting on Alice (computer offline)".

On the ticket, the rail's **Plays** section shows what is waiting: each queued run with its person and why it waits (computer offline, busy with another run, ticket busy, paused, or what to fix in their computer setup), and **Cancel** for that person or anyone with `autoplays:write`. When several computers are paired and none is the default, the person it runs on gets **Pick a computer**: a menu of their computers with each one's T3 projects. Picking one starts the run there and saves it as their link for the project, as the run dialog does the first time, so their other runs in that project start without asking; it doesn't change their default. While the run still waits, say because the picked computer went offline, the same person gets **Change computer** to pick another, which replaces the pick and the link. Once the run is starting, a pick is refused with "the run is already starting". Once a ticket reaches the daily cap, it shows **Auto plays paused** with the runs today; the ticket's developer or anyone with `autoplays:write` can press **Resume**, or it resumes by itself as the day rolls on. What already happened sits in the ticket's thread at the time it happened: "Fix with AI skipped: no longer unblocked" when the ticket stopped matching before the run's turn came, or "Fix with AI didn't run" with the reason when there was nobody to run it on. **Run it** on the latest one runs the play on your own computer. A ticket with no thread yet shows these lines above **Start thread**. A doc shows the same: queued and paused beside its trail, skipped and didn't run in its thread, or beside its trail too while it has no thread.

## Run on a chosen computer

Any auto play can run on one computer you pick, such as a box that is on all day. **Run on** then asks for the computer, and lists the ones you own. A computer shared with you shows as "can't run here yet" and can't be picked yet. A stack has no developer, tester or person who caused a crash, so a stack play always runs this way.

The run is yours: it takes the identity and permissions of whoever picked the computer, shows as their run, and waits in their queue. Saving the auto play again with the same computer keeps who picked it; picking a different computer makes the person who saved it the one it runs as. An auto play that runs on a computer can't be switched on until one is picked.

Nexul checks that you still own the computer or hold **Run agents** on it when you pick it, when the moment happens, and again just before the run starts. If you no longer do, the run doesn't start and shows as **didn't run** with that reason. If someone else has picked the computer since the run was queued, it is skipped.

## Investigate a crash

**Investigate a crash** is a built-in stack play with one auto play, "When a stack **service crashes**", that starts off. A service crashes when its container dies with a non-zero exit code or is killed for running out of memory, outside a deploy; a service that keeps restarting counts once. To switch it on, open **Configuration → Plays**, press **Edit** on **Investigate a crash**, go to the **Auto plays** tab, pick the computer it runs on, and turn it on. You can add conditions on the project, the stack or the service, such as only `api`.

The agent reads the stack with `stack_get`, the crashed service's logs, and the stack's recent deploys, and finds the cause in the project's repository. It writes what it found on the crash's ticket if one is open for that service, or on a new ticket in the stack's project. It never deploys, restarts or edits the stack, and opens a fix pull request only if you edit the play's instructions to say so. If it thinks the fault is in Nexul rather than your app, it says so and points to the [triage skill](/docs/guide/triage/).

A stack run is recorded as a trail under the person whose computer it ran on, in the stack's own thread. One run goes at a time on a stack, and it counts against the same daily limit of automatic runs a ticket has, per stack, so a service that keeps failing can't start a run every time.

## The decisions check

The **Decisions check** is a built-in play with one auto play, so nobody has to press it. When a ticket enters a Done column, the agent reads the ticket and its pull requests and updates the project's [decisions log](/docs/guide/memories/#the-decisions-log): it adds an entry, marks an older one superseded, or leaves the log alone.

Its auto play starts off. To switch it on for a workspace, open **Configuration → Plays**, press **Edit** on **Decisions check**, go to the **Auto plays** tab, and turn on "When a ticket **enters Done**". It runs on the computer of the person who moved the card, or the ticket's developer's when an automation moved it. You can add conditions to it like any auto play, or press the play on a done ticket to run it by hand.

If it can't start, the ticket shows **Decisions check didn't run** with the reason. Press **Run check** to run it on your own computer.

## Who can do what

| Permission | Lets you |
|---|---|
| `plays:read` | see plays: **Configuration → Plays**, their buttons, and their runs |
| `plays:run` | press the plays you can see |
| `plays:write` | create and edit plays, set who may see and run each one, stop anyone's run |
| `plays:delete` | delete plays |
| `autoplays:read` | see a play's auto plays and the daily cap |
| `autoplays:write` | add, change, and switch auto plays, and set the daily cap |
| `autoplays:delete` | delete auto plays |

A role gives these for every play; a play's own rules turn **See this play**, **Run**, **Edit** and **Delete** on or off for that play alone. A play whose own rule allows you **Run** shows its button even when your roles don't hold `plays:run`, and one whose rule denies it keeps its button hidden.

Agents create a play of any type with `play_create`, and add and change its auto plays, including the computer a run goes to with `computer_id`, with `play_update`. They run plays with `play_run` and read trails with `trail_list`. See [MCP server](/docs/guide/mcp-server/). Like the dialog, `play_run` asks where the first time: in a project you haven't linked it needs `computer_id` and `t3_project_id`, and saves them as your link once that computer lists the T3 project. `computer_list` with a computer's id lists its T3 projects.

Auto plays, automation `runPlay` requests, and the automatic decisions check use your project link or, without one, your defaults. The decisions check's **Run again** keeps the same fallback. These paths have no location dialog.
