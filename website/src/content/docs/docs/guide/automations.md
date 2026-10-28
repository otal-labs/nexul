---
title: Automations
description: Run your own TypeScript when something happens in a workspace, such as moving a ticket when its pull request merges.
sidebar:
  order: 12
---

An automation is code that runs when an event happens: a ticket is created, a pull request opens, a deploy fails. There's no rule builder. You write a handler in TypeScript against the Nexul SDK, push it, and Nexul runs it.

Each automation belongs to one workspace. It hears that workspace's events, plus instance-wide ones such as runners and DNS, and you switch and configure it on that workspace's **Automations** page. It never hears a direct message or a private channel.

## The defaults

Every workspace comes with its own copy of three. The first two are switched on:

- **Ticket finished** moves a ticket to a status you choose once all its pull requests have merged.
- **PR opened** moves a ticket to a status you choose when a linked pull request opens.

Open each one's **Configuration** tab and pick the status. Until you do, it skips every event and logs why on its **Runs** tab.

The third, **Crash intake**, starts switched off. It turns crash reports a [bot](/docs/guide/bots/) posts into bug tickets, described below.

### Crash intake

Your app reports a crash by posting it through a bot's URL in the crash-report/1 format: the line `Crash fingerprint: <hash>` as the message, and one embed whose fields are Contract (`crash-report/1`), Fingerprint, Kind, Process type, App version, Build commit, OS, Arch, First seen, Signature, Stack, Truncated, and optionally Suppressed duplicates. Any other post is ignored.

Switch it on, then on its **Configuration** tab pick the channel the bot posts in and the project its tickets go to. Until you do, it skips every bot post and logs why. The bug type (`bug`) and the high-priority label (`high-priority`) can be changed there too. For each report, it looks for a ticket in that project whose body has the report's fingerprint line:

- **None:** it files a bug in the first column, with the fingerprint line, the kind, the first version, the OS, the signature and the stack. Its origin is marked unknown.
- **An open ticket:** it keeps one **Seen again** note on it, updated in place with the number of reports, the versions and when it was last seen. It never files a second ticket.
- **A done ticket with no merged pull request**, such as one closed as won't fix: the same **Seen again** note, and nothing else.
- **A done ticket with merged pull requests:** it asks GitHub whether the report's build commit contains every one of them. If it does, the crash came back, so it files a new bug found in the done ticket, labels it `high-priority`, and puts it at the top of the column. If the build lacks a fix, it updates an **Old build** note instead. If GitHub can't tell (the repository isn't connected, it doesn't know the commit, or it didn't answer), it updates a **Can't tell** note that says why, and files nothing.

The notes are posted on behalf of the workspace's Owner. Each report's **Runs** entry logs what it did.

The **Decisions check** isn't here: it's a [play](/docs/guide/plays/#the-decisions-check) with an auto play, switched on from the play.

## Write your own

1. On the **Automations** page, press **New automation**. Name it and pick its **Scopes**: what its token may read and change, from the same [permissions](/docs/guide/api-and-tokens/) integrations use. The dialog then shows the commands for the next steps, with your instance URL and the automation's id filled in.
2. In an empty folder, run `bun add @otal-labs/nexul-sdk`; its commands need Bun. Then run `bunx nexul init --url <your instance URL> --id <automation-id>`. It asks for a personal access token from **Settings → Security**, keeps both in `nexul.config.json` (added to `.gitignore`, since it holds the token), and writes `src/index.ts`, named after the automation, and a `tsconfig.json` so your editor knows the SDK's types. The name and description in your code are the ones Nexul shows once it runs.
3. Write your handlers:

   ```ts
   import { defineAutomation } from "@otal-labs/nexul-sdk/automation";

   const automation = defineAutomation({
     name: "my-automation",
     description: "Describe what this automation does.",
     config: {},
   });

   automation.on("ticket.created", async (payload, ctx) => {
     ctx.log("ticket created", { id: payload.ticket.id });
     return true;
   });

   export default automation;
   ```

   Return `true` for success. A thrown error, a timeout (30 seconds by default), or any other value counts as a failed run. `payload` is typed from the event catalog, and a list in it is always an array, `[]` when empty. `ctx` gives you `ctx.api`, a typed client acting with the automation's token, plus `ctx.config`, `ctx.secrets`, `ctx.log`, and `ctx.runPlay` ([Start a play](#start-a-play)).
4. Run `bunx nexul dev`, pick a topic, and it fires a sample event at your handler. You see the outcome, the logs, and every API call it would have made. Nothing reaches your instance.
5. Run `bunx nexul push <automation-id> "message"` to upload the code as a pending version.
6. On the automation's **Versions** tab, read the diff against the active code and press **Merge**. The automation restarts on the new code.
7. A new automation starts switched off. Switch it on at the top of its page, and its next event runs your code; each run shows on the **Runs** tab with its logs.

Nexul keeps your `src/index.ts` beside the bundle it runs, so the **Versions** diff compares what you wrote, and `bunx nexul pull <automation-id>` writes the active version's `src/index.ts` back. Other files you import go into the bundle only, and a version pushed before an SDK that keeps sources (0.1.2) has nothing to pull; its diff shows the bundled code. Run these commands with `bunx` from the project folder: a bare `nexul` is the command that installs and runs your instance, not the SDK's.

For unit tests, `createMockContext` from `@otal-labs/nexul-sdk/testing` gives you the same recording context `nexul dev` uses. It records `ctx.runPlay` as the `POST /api/plays/queue` call it makes and answers with a queued run, unless your test scripts another answer under that key.

Two events mark a moment rather than one change. `ticket.unblocked` fires when a ticket's last open blocker gets its pull request merged, reaches a done column, has its blocked-by link removed, or is deleted, with `cause` saying which (`blocker_finished`, `blocker_done`, `link_deleted`, `blocker_deleted`). `doc.settled` fires once a person's edits to a doc have stopped for ten minutes, with `first` true for a doc created inside that window; edits an agent makes through MCP, or an automation makes, never start the wait.

## When a service goes down

`service.container_stopped` fires when a service of a stack goes down: its container died with a non-zero exit code or was killed for running out of memory. `service.container_recovered` fires once that container has stayed running for a minute. A container that keeps restarting is one outage, so the stopped event fires once, at the first crash. Both carry the stack, service, container, machine, an `outage_id` that pairs the two, the exit code, `oom` and `restart_count`; they carry no output lines, so to see why it crashed, read that service's logs while Docker still holds the container. Events for a stack with a deploy running, and for containers that exit with code 0, are never sent.

## When the error rate trips

`stack.error_rate_crossed` fires when a stack's [error-rate rule](/docs/guide/stacks-and-deploys/#error-rate) trips: the share of 5xx answers went above its limit with enough requests. `stack.error_rate_recovered` fires once every minute of a window has stayed under the limit. Both carry the stack, an `episode_id` that pairs the two, the worst share the episode reached (`peak_percent`) with its `requests` and `errors`, and the rule's `limit_percent`, `window_minutes` and `min_requests` as they were when it tripped. They carry no request lines, since Nexul keeps none. Switching the rule off ends an episode without sending the recovery.

## Start a play

`ctx.runPlay` queues one of the workspace's ticket plays on a ticket, named by the play's label:

```ts
automation.on("ticket.test_failed", async (payload, ctx) => {
  const run = await ctx.runPlay("Fix with AI", payload.ticket.id, { priority: "high" });
  ctx.log("fix queued", { state: run.state, reason: run.reason });
  return true;
});
```

- The run waits in a person's queue exactly like an [auto play](/docs/guide/plays/#auto-plays)'s: on the ticket's developer, or its tester with `runOn: "tester"`, at `priority` `"high"`, `"normal"` (the default) or `"low"`. It counts toward the ticket's daily cap of automatic runs, and the play's show-when stage doesn't apply.
- It answers with `{ id, state, reason }`. `state` is `"queued"`, with `reason` `"paused"` when the ticket has reached its daily cap, or `"didnt_run"` with why, such as the ticket having no developer. Either way the ticket shows it.
- The automation's token needs the `plays:run` scope, and whoever created the automation must be able to run that play on that ticket. The default automations have no creator, so they can't start plays.

Play names are checked as you type. Run `bunx nexul types <automation-id>` and it writes `src/plays.generated.d.ts` with the workspace's play names, so a typo, a doc play, or a play renamed since then is a type error in your editor. `nexul push` refreshes the file every time. A project made by `nexul init` already points `src/index.ts` at it; in an older project, add `/// <reference path="./plays.generated.d.ts" />` as the first line of `src/index.ts`. Labels are unique in a workspace, so a name is always one play.

Code you already pushed keeps the name it was written with. Renaming a play warns first: the play dialog lists the automations whose active or pending code still has the old label in quotes and asks before it saves, and `play_update` over MCP renames it and names them in its `warnings`. If someone renames or deletes that play anyway, `ctx.runPlay` throws, nothing is queued, and the run fails with the server's reason on the **Runs** tab, such as `no play named "Fix with AI" in this workspace`. Rename the play back, or update the name, push, and merge.

## Versions

Every push is kept as a version with its author, time, and message, and lands pending: a push never changes running code by itself. **Rollback** on an older version makes it active again.

## Secrets

Set secrets on the **Secrets** tab of the Automations page. Every automation in the workspace reads them as `ctx.secrets.NAME`; no other workspace sees them. Once saved, a value can be replaced (**Replace** on its row) or deleted but never read back.

## Where automations run

Automations run on an automations host. `nexul install` puts one on your instance's server, named `instance`, and new automations go there.

To run automations on another machine, for example to reach services on its network:

1. On the **Hosts** tab, press **Add automations host**.
2. Give it a **Host name** (lower-case letters, digits, and dashes) and the **Machine** it runs on. **Add automations host** appears when you hold **Manage** on a machine, and the Machine field offers only the machines you manage. The instance Owner can also type a machine Nexul doesn't know yet, or leave it empty, since a host doesn't add machines.
3. Run the install line shown, for Linux or macOS, or for Windows, on that machine. It needs no Docker. The line carries a one-time code that expires after an hour.

A host belongs to its machine, like a runner. You see it when you can see its machine, and its **Remove** button appears where you hold **Manage** on the machine; a host on a machine you can't see is not found. Placing an automation on a host takes **Deploy to** on the host's machine as well as `automations:write` in the automation's workspace, and the instance host is open to every automation. Taking someone off a machine leaves the automations already placed there running; their next move is checked. Agents add and remove hosts with `host_create` and `host_delete` and `kind: "automations"`, and place an automation with `automation_update` and `host_id`.

Move an automation by picking another host in **Runs on** on its page. The list holds the instance host and the hosts on machines you hold **Deploy to** on. It stops on the old host as it starts on the new one, so two hosts never run the same automation. To retire a host, move its automations off, then **Remove** it; it uninstalls itself. Removing a host moves its automations to the instance host and stops its credential. Adding a host again makes a new one: its automations stay on the instance host until you move them back.

## Switching off

Switch an automation off and it receives nothing until you switch it on again. Events in between are skipped, not replayed. A host that loses its connection is different: it picks up where it left off when it reconnects.

Its token stops working too: every API call with it is refused with `automation is switched off`, and its host stops running it. Switch it back on and the same token works again; there's nothing to mint.
