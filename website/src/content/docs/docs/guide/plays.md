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
4. Check **Where to run**. It names the first of [your computers](/docs/guide/paired-computers/#your-computers-in-order) that can take the run, and any it skips with why, such as "Skipping: laptop is offline". **Change** picks another of your computers and its T3 project for this one run; your order stays as it is, and a checkout you pick is remembered for the project. When none of your computers has the project yet, the dialog asks for a computer and a T3 project, and remembers the pick the same way.
5. Check the provider and model on the pill under it. They come from the project's settings or your defaults; change them here for this run only.
6. Start the run.

A run keeps the target's T3 thread unless that thread is in another T3 project, which starts a fresh one; picking the same place again keeps it. Nexul uses a picked T3 project only once the computer lists it. Once a run has started it stays where it started: an answer to its question, **Continue this run**, and following it again after a Nexul restart all go to its own thread, even if you change your order or your project's computers meanwhile.

If the play can't run, the dialog says why instead: no computer paired, a pairing that expired, or none of your computers able to take it, with the reason for each. A pressed run never waits for a computer. A play on a blocked ticket asks you to confirm first.

The agent decides where the ticket goes when it finishes. It reads your footer memories for that, so a footer memory that says "move the card to Review when the pull request is open" is how you get cards to move.

## Follow a run

While a play runs, its button shows a spinner and opens the run; the other plays wait, greyed out. Every press leaves a trail, listed in the **Trail** section of the ticket, doc, or Interview page. A trail shows to whoever started the run and to everyone who can see the play. Open one to see who started it, what was picked, and a transcript of what the agent said and did. While the agent works, one line names the step it is on, such as "Running bun run test". Each stretch of work then folds to a line such as "Worked for 14s · explored 2 files, ran 1 command". Open the fold to see the files it read or edited, the commands it ran (neighbours listed under one count, without the `cd` into the run's own folder), and any other tool it called. Click a file name, a command, or a tool to see its arguments and result.

While a run is going:

- **The agent asks you something.** A question card shows on the trail and in the thread, and a dot on the ticket's **Thread** tab marks it until it is answered. Answer it and the same run carries on. You can answer in T3 Code instead, picking an option or typing your own; the card then shows that answer. A secret the agent asks for, such as an API key, gets a secret card in the thread the same way; what you save goes to T3 Code, never into the thread.
- **You want it to stop.** Press the stop square beside the play's button. You, or anyone who manages plays, can stop it, even while it waits on a question.
- **The computer drops off.** The trail shows **Reconnecting to T3 Code…** and Nexul keeps trying. The run fails only if the computer stays unreachable for 15 minutes; until then it keeps its place in the computer's **Runs at once**.
- **Nothing happens for 15 minutes.** The run fails and its turn in T3 Code is stopped.

When a run has ended, type into **Continue this run** to send the agent its next step on the same thread. The play's instructions are not sent again. Mentioning **@Agent** in the target's thread does the same once its newest run has ended, so the board card shows the run going again; while a run is still going, or if you can't steer it, the mention gets a plain agent reply instead. If you keep working on the thread in T3 Code itself, the trail picks up those turns while you have Nexul open. If the thread is gone from T3 Code, the play starts again as a new run in the same place, with your message added to its instructions. When Nexul doesn't know where an older run ran, the run dialog opens on **Where to run** instead, with your message filled in.

A doc play locks its doc when it starts, so nobody edits the doc under the agent, and the doc stays locked afterwards. Someone with the lock permission unlocks it. **Clarify via AI** and **Update doc via AI** are the exceptions: they unlock the doc when their run ends.

## The built-in plays

Every workspace starts with ten. They are ordinary plays: rename, edit, or delete them.

| Play | Runs on | What it does |
|---|---|---|
| **Fix with AI** | a ticket in an In progress column | Reads the ticket, fixes it on its own branch, and opens a pull request with the computer's own GitHub sign-in. |
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

Open **Plays** in the sidebar. Plays are grouped by type, and a row says the stage a ticket play shows on, when it starts by itself, and when it is off. Press **+** beside the list's title to make one, or open a play and press **Edit** to change it. **Clone** in a play's row menu opens a new play filled in from it and named with " copy" after its label. On a built-in play, it copies the instructions onto a [template](/docs/guide/memories/#templates) instead. A play has:

- **Label**, the button text. Each play in a workspace has its own label, ignoring case, since an automation names a play by it; a label already in use is refused at the field. Renaming a play that automations still call by its old label with `runPlay` asks first and names them, since they fail until their code uses the new label; **Rename anyway** saves it.
- **Type**: ticket, doc, interview, or stack. It can't change later.
- **Show when**, the one board stage a ticket play shows in.
- **Description**, shown as the button's tooltip.
- **Instructions**, what the agent should do on every run.
- **Enabled**, and **Excluded projects** where it never shows. The **Enabled** switch also sits beside the tabs on the play's page and saves as soon as you flip it.

A built-in play says whether its instructions match the instance's template, with **Reset to instance template** when they don't. See [Templates](/docs/guide/memories/#templates).

To choose who may see, run, edit or delete one play, right-click its row in the list, or open its **⋯** menu on its page, pick **Permissions…**, and set **See this play**, **Run**, **Edit** or **Delete** to **Allow** or **Deny** for a role or a person; see [Permissions on a project, a folder, a doc or a play](/docs/guide/people-and-access/#permissions-on-a-project-a-folder-a-doc-or-a-play). Running a play needs seeing it: for a role or person denied **See this play**, the **Run** row keeps its own answer and adds a line such as "Can't run while See this play is denied · Denied on play Deploy prod for role Client".

A play someone can't see is gone for them: from **Plays**, the buttons and menus on tickets, docs and the Interview page, its auto plays and queued runs, and agents' `play_list`, and its link is not found. Its auto plays never run on them. Runs it already made stay readable to whoever started them and to everyone who can see the play.

## Auto plays

An auto play starts its play by itself when something happens to a ticket, a doc, or a stack, so nobody has to press the button. A play can have several. Open the play in **Plays** and go to its **Auto plays** tab. Interview plays have none.

Each auto play reads as one sentence, such as "When a ticket **becomes unblocked**, if **Type is Bug** → **High**, else **Normal**, runs on **Developer**", with a switch to turn it on or off. A new one starts switched off. Press a row to change it, or use its menu to duplicate or delete it. Switching one off, or deleting it, skips the runs it still has queued.

An auto play has:

- **When**, the moment it waits for. A ticket play: the ticket becomes unblocked, enters a stage, is created, gets a developer, gets a tester, or fails a test. Setting the developer or tester after the ticket entered a stage counts as entering it again for that stage's auto plays when the change is what lets them run: their conditions now match, or they now have someone to run on. A doc play: the doc is created, or changes. A change counts once edits have stopped for 10 minutes, and never for an agent's edits. A stack play: a service crashes, recovers once it has run steadily again, or the stack's share of 5xx answers goes over its [error-rate rule](/docs/guide/stacks-and-deploys/#error-rate). The last names no service, so a condition on the service never matches it, and the run is told the stack and reads its open episode with `stack_get`.
- **If**, conditions on the ticket's or doc's fields: type, project, stage, status column, category, label, developer, tester, whether it has a source doc or a linked pull request, and whether it is blocked; a doc's project and folder; a stack's project, the stack, and the service by name. Pick whether all or any of them must match, and add a group one level in for an "any of these" inside an "all of these". With no conditions, every match of the moment runs it.
- **Priority**, High, Normal, or Low, with rules such as "High if Type is Bug" checked in order, then the level for everything else. A higher priority runs first in the person's queue.
- **Limits**, at most once per ticket or doc every hour, 24 hours, or 7 days, or no limit.
- **Run on**, whose computer runs it: the ticket's developer, its tester, whoever caused the moment, your computers, or picked computers. A doc play has no developer or tester, so it offers whoever caused it, your computers, and picked computers. A stack play offers only your computers and picked computers. See [Run on a chosen computer](#run-on-a-chosen-computer).
- **Memories**, extra memories each run reads, beside the ones always included, picked the way the run dialog picks them. The list holds every project's memories, each named with its project, and a run reads only the picked ones from its own ticket's, doc's, or stack's project. Deleting a memory takes it off every auto play that picked it.

The play's **Show when** stage does not apply to an auto play; add a **Stage is** condition instead. A run waits until one of its person's computers can take it, and its conditions are checked again before it starts. If nothing is paired yet, a computer needs pairing again, none of their computers holds the project, or none is online, set up, and free, the run waits rather than failing and starts once that is fixed, on whichever computer can take it then. The person it runs on needs to see the play and hold `plays:run`; if they can't see it, it doesn't run, and says why only to those who can.

**Each ticket runs at most 5 auto plays a day** across every play, so two auto plays can't keep starting each other. Change the number under the list of any ticket play's auto plays; it applies to the whole workspace.

**Each computer takes 1 run at a time** by default, pressed or automatic. Set its **Runs at once**, up to 5, in the computer's details under [Settings → Computers](/docs/guide/paired-computers/#your-computers-in-order), or right under the cap line on a play's auto plays. A full computer is passed over for the next one in its person's order, so an auto run waits ("computers busy") only when every computer that could take it is full, and starts as soon as one has room. Runs side by side work best when new threads start in their own worktree. The **At most once per ticket every** limit on an auto play is a different thing: how soon the same ticket may start that auto play again.

Under the list, "Right now" says how many runs of the play are queued and whom they wait on, such as "1 waiting on Alice (computers busy)".

On the ticket, the rail's **Plays** section shows what is waiting: each queued run with its person and why it waits (no computer can take it, computers busy, ticket busy, paused, or what to fix in their computer setup), and **Cancel** for that person or anyone with `autoplays:write`. The person it runs on also sees the reason for each of their computers, such as "laptop is offline; build-box is full (2 of 2 running)"; nobody else does. On the board, the ticket's card shows a clock beside its key while a run waits, until the run starts and the spinner takes its place. Once a ticket reaches the daily cap, it shows **Auto plays paused** with the runs today; the ticket's developer or anyone with `autoplays:write` can press **Resume**, or it resumes by itself as the day rolls on. What already happened sits in the ticket's thread at the time it happened: "Fix with AI skipped: no longer unblocked" when the ticket stopped matching before the run's turn came, or "Fix with AI didn't run" with the reason when there was nobody to run it on. **Run it** on the latest one runs the play on your own computer. A ticket with no thread yet shows these lines above **Start thread**. A doc shows the same: queued and paused beside its trail, skipped and didn't run in its thread, or beside its trail too while it has no thread.

## Run on a chosen computer

Any auto play can run on computers you pick, such as a box that is on all day with a laptop behind it. Choose **Your computers** or **Picked computers** under **Run on**. **Picked computers** starts as your own computers in their order, and lists only the ones you own. A computer shared with you shows as "can't run here yet" and can't be picked yet. A stack has no developer, tester or person who caused a crash, so a stack play always runs this way.

Drag a computer by its grip to put the ones you want tried first at the top, press its **×** to take it out, and use **Add computer** at the end of the list to put another back. Each row shows how many runs the computer has going. The run goes to the first one that is online, set up, holds the project and has room, so a box that is shut or full sends the run to the next one in your list. One computer is a list of one. Pick **Your computers** instead to use your own computers as they are ordered under [Settings → Computers](/docs/guide/paired-computers/#your-computers-in-order), so the auto play follows that list as you change it.

The run is yours: it takes the identity and permissions of whoever picked the computers, shows as their run, and waits in their queue. Under the list, the composer says who that is, such as **Runs as you.** Saving the auto play again with the same list keeps who picked it; saving a different list or order makes the person who saved it the one it runs as. Saving the auto play with a new list moves the runs already waiting onto it, so a run held on a full computer is tried on the ones you added at once. An auto play that runs on computers can't be switched on until they are picked.

Nexul checks that you still own each computer or hold **Run agents** on it when you pick it, when the moment happens, and again just before the run starts. A computer you no longer have the right on is left out. If that leaves none, the run doesn't start and shows as **didn't run** with that reason.

## Investigate a crash

**Investigate a crash** is a built-in stack play with one auto play, "When a stack **service crashes**", that starts off. A service crashes when its container dies with a non-zero exit code or is killed for running out of memory, outside a deploy; a service that keeps restarting counts once. To switch it on, open **Plays**, open **Investigate a crash**, go to its **Auto plays** tab, pick the computers it runs on, and turn it on. You can add conditions on the project, the stack or the service, such as only `api`.

The agent reads the stack with `stack_get`, the crashed service's logs, and the stack's recent deploys, and finds the cause in the project's repository. It writes what it found on the crash's ticket if one is open for that service, or on a new ticket in the stack's project. It never deploys, restarts or edits the stack, and opens a fix pull request only if you edit the play's instructions to say so. If it thinks the fault is in Nexul rather than your app, it says so and points to the [triage skill](/docs/guide/triage/).

A stack run is recorded as a trail under the person whose computer it ran on, in the stack's own thread. One run goes at a time on a stack, and it counts against the same daily limit of automatic runs a ticket has, per stack, so a service that keeps failing can't start a run every time.

## The decisions check

The **Decisions check** is a built-in play with one auto play, so nobody has to press it. When a ticket enters a Done column, the agent reads the ticket and its pull requests and updates the project's [decisions log](/docs/guide/memories/#the-decisions-log): it adds an entry, marks an older one superseded, or leaves the log alone.

Its auto play starts off. To switch it on for a workspace, open **Plays**, open **Decisions check**, go to its **Auto plays** tab, and turn on "When a ticket **enters Done**". It runs on the computer of the person who moved the card, or the ticket's developer's when an automation moved it. You can add conditions to it like any auto play, or press the play on a done ticket to run it by hand.

If it can't start, the ticket shows **Decisions check didn't run** with the reason. Press **Run check** to run it on your own computer.

## History

A play keeps the last ten versions of its instructions, and **History** in the play page's ⋯ menu shows them. Each version is compared word by word with the one before it, with a list of versions on the right. An unedited built-in play reads **Built-in default**. Changes to its label, description, enabled switch, stage and excluded projects sit in the same list as single lines ("Enabled On → Off"); they show what happened and can't be picked.

**Restore** on an older version saves its text again as the newest, so nothing in the history is lost. It takes `plays:write`; anyone who can read the play can open its history.

A run's details say **Ran with version 7** and open the history on that version, so you can see the instructions the agent was given. Once ten newer versions have pushed it out, the line reads **Version 7 is no longer kept**. Runs from before history was kept show no version.

## Who can do what

| Permission | Lets you |
|---|---|
| `plays:read` | see plays: **Plays** in the sidebar, their buttons, and their runs |
| `plays:run` | press the plays you can see |
| `plays:write` | create and edit plays, set who may see and run each one, stop anyone's run |
| `plays:delete` | delete plays |
| `autoplays:read` | see a play's auto plays and the daily cap |
| `autoplays:write` | add, change, and switch auto plays, and set the daily cap |
| `autoplays:delete` | delete auto plays |

A role gives these for every play; a play's own rules turn **See this play**, **Run**, **Edit** and **Delete** on or off for that play alone. A play whose own rule allows you **Run** shows its button even when your roles don't hold `plays:run`, and one whose rule denies it keeps its button hidden.

Agents create a play of any type with `play_create`, and add and change its auto plays, including the computers a run goes to with `computer_ids`, in order (an empty list is your own order), with `play_update`. They run plays with `play_run` and read trails with `trail_list`. See [MCP server](/docs/guide/mcp-server/). `play_run` goes to the first of your computers that can take the run, like the dialog. Give `computer_id` and `t3_project_id` to run it on one computer instead, which you must do when none of your computers has the project; the T3 project is remembered as that computer's checkout of the project once the computer lists it. `computer_list` with a computer's id lists its T3 projects.

Auto plays, automation `runPlay` requests, and the automatic decisions check, with its **Run again**, go to the first of the person's computers that can take them: the project's own computers when it names them, otherwise their order. These paths have no location dialog.
